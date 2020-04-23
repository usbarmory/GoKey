# https://github.com/f-secure-foundry/GoKey
#
# Copyright (c) F-Secure Corporation
# https://foundry.f-secure.com
#
# Use of this source code is governed by the license
# that can be found in the LICENSE file.

BUILD_USER = $(shell whoami)
BUILD_HOST = $(shell hostname)
BUILD_DATE = $(shell /bin/date -u "+%Y-%m-%d %H:%M:%S")
BUILD = ${BUILD_USER}@${BUILD_HOST} on ${BUILD_DATE}
REV = $(shell git rev-parse --short HEAD 2> /dev/null)
PKG = "github.com/f-secure-foundry/GoKey"

APP := gokey_imx6
GOENV := GO_EXTLINK_ENABLED=0 CGO_ENABLED=0 GOOS=tamago GOARM=7 GOARCH=arm
TEXT_START := 0x80010000 # ramStart (defined in imx6/imx6ul/memory.go) + 0x10000
GOFLAGS := -ldflags "-s -w -T $(TEXT_START) -E _rt0_arm_tamago -R 0x1000 -X '${PKG}/internal.Build=${BUILD}' -X '${PKG}/internal.Revision=${REV}'"
QEMU ?= qemu-system-arm -machine mcimx6ul-evk -cpu cortex-a7 -m 512M \
        -nographic -monitor none -serial null -serial stdio -net none \
        -semihosting -d unimp

SHELL = /bin/bash
UBOOT_VER=2019.07
LOSETUP_DEV=$(shell /sbin/losetup -f)
DISK_SIZE = 20MiB
JOBS=2
# microSD: 0, eMMC: 1
BOOTDEV ?= 0
BOOTCOMMAND = ext2load mmc $(BOOTDEV):1 0x90000000 $(APP); bootelf -p 0x90000000

.PHONY: clean qemu qemu-gdb

#### primary targets ####

all: $(APP)

u-boot: u-boot-${UBOOT_VER}/u-boot-dtb.imx

u-boot-verified: u-boot-${UBOOT_VER}/usbarmory.itb

elf: $(APP)

raw: $(APP) u-boot $(APP).raw

raw_signed: $(APP) u-boot-verified sign-u-boot $(APP)_signed.raw

#### utilities ####

check_tamago:
	@if [ "${TAMAGO}" == "" ] || [ ! -f "${TAMAGO}" ]; then \
		echo 'You need to set the TAMAGO variable to a compiled version of https://github.com/f-secure-foundry/tamago-go'; \
		exit 1; \
	fi

check_usbarmory_git:
	@if [ "${USBARMORY_GIT}" == "" ]; then \
		echo 'You need to set the USBARMORY_GIT variable to the path of a clone of'; \
		echo '  https://github.com/f-secure-foundry/usbarmory'; \
		exit 1; \
	fi

check_hab_keys:
	@if [ "${KEYS_PATH}" == "" ]; then \
		echo 'You need to set the KEYS_PATH variable to the path of secure/verified boot keys'; \
		echo 'See https://github.com/f-secure-foundry/usbarmory/wiki/Secure-boot-(Mk-II)'; \
		exit 1; \
	fi

check_bundled_keys:
	@if [ "${PGP_SECRET_KEY}" == "" ] || [ ! -f "${PGP_SECRET_KEY}" ]; then \
		echo 'You need to set the PGP_SECRET_KEY variable to the path of a valid PGP secret key'; \
		exit 1; \
	fi

check: # requires honnef.co/go/tools/cmd/staticcheck
	go vet ./...
	${GOPATH}/bin/staticcheck ./...

errcheck: # requires github.com/kisielk/errcheck
	${GOPATH}/bin/errcheck -ignore 'fmt:Fprint*' ./...

clean:
	rm -f $(APP) gokey_vpcd
	@rm -fr $(APP).raw $(APP)_signed.raw u-boot-${UBOOT_VER}* tamago.elf

#### dependencies ####

$(APP): check_tamago check_bundled_keys
	cd $(CURDIR) && ${TAMAGO} generate
	$(GOENV) $(TAMAGO) build $(GOFLAGS) -o $(CURDIR)/$(APP) || (rm -f $(CURDIR)/tmp.go && exit 1)
	rm -f $(CURDIR)/tmp.go

gokey_vpcd: check_bundled_keys
	cd $(CURDIR) && go generate
	go build -tags vpcd -o $(CURDIR)/gokey_vpcd || (rm -f $(CURDIR)/tmp.go && exit 1)
	rm -f $(CURDIR)/tmp.go

qemu: $(APP)
	$(QEMU) -kernel $(APP)

qemu-gdb: $(APP)
	$(QEMU) -kernel $(APP) -S -s

$(APP).raw: $(APP)
	@if [ ! -f "$(APP).raw" ]; then \
		truncate -s $(DISK_SIZE) $(APP).raw && \
		sudo /sbin/parted $(APP).raw --script mklabel msdos && \
		sudo /sbin/parted $(APP).raw --script mkpart primary ext4 5M 100% && \
		sudo /sbin/losetup $(LOSETUP_DEV) $(APP).raw -o 5242880 --sizelimit $(DISK_SIZE) && \
		sudo /sbin/mkfs.ext4 -F $(LOSETUP_DEV) && \
		sudo /sbin/losetup -d $(LOSETUP_DEV) && \
		mkdir -p rootfs && \
		sudo mount -o loop,offset=5242880 -t ext4 $(APP).raw rootfs/ && \
		sudo cp ${APP} rootfs/ && \
		sudo umount rootfs && \
		sudo dd if=u-boot-${UBOOT_VER}/u-boot-dtb.imx of=$(APP).raw bs=512 seek=2 conv=fsync conv=notrunc && \
		rmdir rootfs; \
	fi

$(APP)_signed.raw: $(APP) u-boot-${UBOOT_VER}/usbarmory.itb
	@if [ ! -f "$(APP)_signed.raw" ]; then \
		truncate -s $(DISK_SIZE) $(APP)_signed.raw && \
		sudo /sbin/parted $(APP)_signed.raw --script mklabel msdos && \
		sudo /sbin/parted $(APP)_signed.raw --script mkpart primary ext4 5M 100% && \
		sudo /sbin/losetup $(LOSETUP_DEV) $(APP)_signed.raw -o 5242880 --sizelimit $(DISK_SIZE) && \
		sudo /sbin/mkfs.ext4 -F $(LOSETUP_DEV) && \
		sudo /sbin/losetup -d $(LOSETUP_DEV) && \
		mkdir -p rootfs && \
		sudo mount -o loop,offset=5242880 -t ext4 $(APP)_signed.raw rootfs/ && \
		sudo mkdir -p rootfs/boot && \
		sudo cp u-boot-${UBOOT_VER}/usbarmory.itb rootfs/boot && \
		sudo umount rootfs && \
		sudo dd if=u-boot-${UBOOT_VER}/u-boot-signed.imx of=$(APP)_signed.raw bs=512 seek=2 conv=fsync conv=notrunc && \
		rmdir rootfs; \
	fi

u-boot-${UBOOT_VER}: check_usbarmory_git
	wget ftp://ftp.denx.de/pub/u-boot/u-boot-${UBOOT_VER}.tar.bz2 -O u-boot-${UBOOT_VER}.tar.bz2
	wget ftp://ftp.denx.de/pub/u-boot/u-boot-${UBOOT_VER}.tar.bz2.sig -O u-boot-${UBOOT_VER}.tar.bz2.sig
	gpg --verify u-boot-${UBOOT_VER}.tar.bz2.sig
	tar xf u-boot-${UBOOT_VER}.tar.bz2
	cd u-boot-${UBOOT_VER} && \
		patch -p1 < ${USBARMORY_GIT}/software/u-boot/0001-ARM-mx6-add-support-for-USB-armory-Mk-II-board.patch && \
		patch -p1 < ${USBARMORY_GIT}/software/u-boot/0001-Drop-linker-generated-array-creation-when-CONFIG_CMD.patch

u-boot-${UBOOT_VER}/u-boot-dtb.imx: u-boot-${UBOOT_VER}
	cd u-boot-${UBOOT_VER} && \
		make distclean && \
		make usbarmory-mark-two_defconfig && \
		sed -i -e 's/run start_normal/${BOOTCOMMAND}/' include/configs/usbarmory-mark-two.h && \
		CROSS_COMPILE=arm-linux-gnueabihf- ARCH=arm make -j${JOBS}

u-boot-${UBOOT_VER}/usbarmory.itb: check_usbarmory_git check_hab_keys u-boot-${UBOOT_VER}
	ln -s gokey_imx6 tamago.elf
	cd u-boot-${UBOOT_VER} && \
		patch -p1 < ${USBARMORY_GIT}/software/u-boot/0002-Add-USB-armory-mark-two-tamago-fit-image-boot.patch
	@if [ ${BOOTDEV} == 1 ]; then \
		sed -i -e 's/CONFIG_SYS_BOOT_DEV_MICROSD=y/# CONFIG_SYS_BOOT_DEV_MICROSD is not set/' u-boot-${UBOOT_VER}/configs/usbarmory-mark-two_defconfig; \
		sed -i -e 's/# CONFIG_SYS_BOOT_DEV_EMMC is not set/CONFIG_SYS_BOOT_DEV_EMMC=y/' u-boot-${UBOOT_VER}/configs/usbarmory-mark-two_defconfig; \
	fi
	sed -i -e 's/CONFIG_SYS_BOOT_MODE_NORMAL=y/# CONFIG_SYS_BOOT_MODE_NORMAL is not set/' u-boot-${UBOOT_VER}/configs/usbarmory-mark-two_defconfig
	sed -i -e 's/# CONFIG_SYS_BOOT_MODE_VERIFIED_LOCKED is not set/CONFIG_SYS_BOOT_MODE_VERIFIED_LOCKED=y/' u-boot-${UBOOT_VER}/configs/usbarmory-mark-two_defconfig
	export CROSS_COMPILE=arm-linux-gnueabihf- && cd u-boot-${UBOOT_VER} && \
		make distclean && \
		make usbarmory-mark-two_defconfig && \
		make -j${JOBS} tools CONFIG_MKIMAGE_DTC_PATH="scripts/dtc/dtc" && \
		make -j${JOBS} dtbs ARCH=arm CONFIG_MKIMAGE_DTC_PATH="scripts/dtc/dtc" && \
		cp ${USBARMORY_GIT}/software/secure_boot/mark-two/usbarmory-tamago.its . && \
		tools/mkimage -D "-I dts -O dtb -p 2000 -i $(CURDIR)" -f usbarmory-tamago.its usbarmory.itb && \
		tools/mkimage -D "-I dts -O dtb -p 2000" -F -k ${KEYS_PATH} -K arch/arm/dts/imx6ull-usbarmory.dtb -r usbarmory.itb && \
		ARCH=arm make -j${JOBS}

sign-u-boot: check_usbarmory_git check_hab_keys u-boot-${UBOOT_VER}/usbarmory.itb
	cd u-boot-${UBOOT_VER} && \
		${USBARMORY_GIT}/software/secure_boot/usbarmory_csftool \
			--csf_key ${KEYS_PATH}/CSF_1_key.pem \
			--csf_crt ${KEYS_PATH}/CSF_1_crt.pem \
			--img_key ${KEYS_PATH}/IMG_1_key.pem \
			--img_crt ${KEYS_PATH}/IMG_1_crt.pem \
			--table   ${KEYS_PATH}/SRK_1_2_3_4_table.bin \
			--index   1 \
			--image   u-boot-dtb.imx \
			--output  csf.bin && \
		cat u-boot-dtb.imx csf.bin > u-boot-signed.imx
