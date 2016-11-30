% DOCKER-LVM-PLUGIN(8) 
% Shishir Mahajan 
% FEBRUARY 2016
# NAME
docker-lvm-plugin - Docker Volume Driver for lvm volumes

# SYNOPSIS
**docker-lvm-plugin**
[**-debug**]
[**-version**]

# DESCRIPTION
This plugin can be used to create lvm volumes of specified size,
which can then be bind mounted into the container using `docker run` 
command. 

# USAGE
Start the docker daemon before starting the docker-lvm-plugin daemon. 
You can start docker daemon using command:
```bash
systemctl start docker 
```
Once docker daemon is up and running, you can start docker-lvm-plugin daemon
using command:
```bash
systemctl start docker-lvm-plugin
``` 
docker-lvm-plugin daemon is on-demand socket activated. Running `docker volume ls` command
will automatically start the daemon.

Since logical volumes (lv's) are based on a volume group, it is the 
responsibility of the user (administrator) to provide a volume group name. 
You can choose an existing volume group name by listing volume groups on 
your system using `vgs` command OR create a new volume group using `vgcreate` 
command. e.g.
```bash
vgcreate vg1 /dev/hda 
```
where /dev/hda is your partition or whole disk on which physical volumes 
were created.

Add this volume group name in the config file. 
```bash
/etc/docker/docker-lvm-plugin
```
The docker-lvm-plugin also supports the creation of thinly-provisioned volumes. To create a thinly-provisioned volume, a user (administrator) must first create a thin pool using the `lvcreate` command.
```bash
lvcreate -L 10G -T vg1/mythinpool
```
This will create a thinpool named `mythinpool` of size 10G under volume group `vg1`.
NOTE: thinpools are special kind of logical volumes carved out of the volume group.
Hence in the above example, to create the thinpool `mythinpool` you must have atleast 10G of freespace in volume group `vg1`.

# OPTIONS
**-debug**=*true*|*false*
  Enable debug logging. Default is false.
**-version**=*true*|*false*
  Print version information and quit. Default is false.

# EXAMPLES
**Volume Creation**
```bash
docker volume create -d lvm --opt size=0.2G foobar
```
This will create a lvm volume named foobar of size 208 MB (0.2 GB).
```bash
docker volume create -d lvm --opt size=0.2G --opt thinpool=mythinpool thin_vol
```
This will create a thinly-provisioned lvm volume named `thin_vol` in mythinpool.
```bash
docker volume create -d lvm --opt snapshot=foobar --opt size=100M foobar_snapshot
```
This will create a snapshot volume of `foobar` named `foobar_snapshot`. For thin snapshots, use the same command above but don't specify a size.
```bash
docker volume create -d lvm --opt size=0.2G --opt keyfile=/root/key.bin crypt_vol
```
This will create a LUKS encrypted lvm volume named `crypt_vol` with the contents of `/root/key.bin` as a binary passphrase. Snapshots of encrypted volumes use the same key file. The key file must be present when the volume is created, and when it is mounted to a container.

**Volume List**
```bash
docker volume ls
```
This will list volumes created by all docker drivers including the default driver (local).

**Volume Inspect**
```bash
docker volume inspect foobar
```
This will inspect foobar and return a JSON.
[
    {
        "Driver": "lvm",
        "Labels": {},
        "Mountpoint": "/var/lib/docker-lvm-plugin/foobar",
        "Name": "foobar",
        "Options": {
            "size": "0.2G"
        },
        "Scope": "local"
    }
]

**Volume Removal**
```bash
docker volume rm foobar
```
This will remove lvm volume foobar.

**Bind Mount lvm volume inside the container**
```bash
docker run -it -v foobar:/home fedora /bin/bash
```
This will bind mount the logical volume foobar into the home directory of the container.



