## Build a TDX compatiable image

### 1. Build linuxkit with Alibaba Cloud support
```sh
git clone https://github.com/yuguorui/linuxkit.git
cd linuxkit
make 

INITRD_LARGE_THAN_4GiB=1
# Original image format is built with fat32, which requires the initrd is less
# than 4 GiB.
#
# If the INITRD is less than 4 GiB, then we can use the original raw-efi image
# format.
if [ $INITRD_LARGE_THAN_4GiB -eq 1 ]; then
    (cd tools/grub && docker build -f Dockerfile.rhel -t linuxkit-hack/grub .)

    if [ -z $(docker ps -f name='registry' -q) ]; then
      docker run -d -p 5000:5000 --restart=always --name registry registry:2
    fi

    (
      remote_registry="localhost:5000/"
      tag="v0.1"
      cd tools/mkimage-raw-efi-ext4/ && 
      docker build . -t ${remote_registry}mkimage-raw-efi-ext4:$tag && 
      docker push ${remote_registry}mkimage-raw-efi-ext4:$tag
    )
    image_format="raw-efi-ext4"
else
    image_format="raw-efi"
fi
```

### 2. Build linux kernel with TDX support
```sh
(
  cd contrib/foreign-kernels && 
  docker build -f Dockerfile.rpm.anolis.5.10 . -t linuxkit/kernel:5.10-tdx
)
```

### 3. Build a raw-efi image with config
```sh
image_format="raw-efi-ext4"
bin/linuxkit build --docker examples/dm-crypt-tdx.yml -f $image_format
```

### 4. Push the image to Alibaba Cloud
```sh
# push linuxkit appimage to Alibaba Cloud
bin/linuxkit push alibabacloud dm-crypt-tdx-efi.img [...other params]
```

### 5. Create TD Instance on Alibaba Cloud Instances with built images