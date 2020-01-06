# Bitmarkd Packer Builder

This is a packer script that helps on build images for bitmarkd.

# Build

Simply run a build by the following command:

```
$ packer build -var 'bitmarkd_version=0.12.0' packer_build.json
```

Make sure you set the version to the one you'd like to build.
