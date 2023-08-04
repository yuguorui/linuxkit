## How to build a TDX compatiable image

```sh
git clone https://github.com/yuguorui/linuxkit.git
cd linuxkit
make 

cd contrib/foreign-kernels && docker build -f Dockerfile.rpm.anolis.5.10 . -t linuxkit/kernel:5.10-tdx
cd ../../ 

# build a raw-efi image
bin/linuxkit build --docker examples/dm-crypt-tdx.yml -f raw-efi

# push image to alibaba cloud
bin/linuxkit push alibabacloud dm-crypt-tdx-efi.img [...other params]
```
