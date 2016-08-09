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
Since logical volumes (lv's) are based on a volume group, it is the 
responsibility of the user (administrator) to provide a volume group name. 
You can choose an existing volume group name by listing volume groups on 
your system using `vgs` command OR create a new volume group using `vgcreate` 
command. e.g.
```bash
vgcreate volume_group_one /dev/hda 
```
where /dev/hda is your partition or whole disk on which physical volumes 
were created.

Add this volume group name in the config file. 
```bash
/etc/docker/docker-lvm-plugin
```
The docker-lvm-plugin also supports the creation of thinly-provisioned volumes. To create a thinly-provisioned volume, a user (administrator) must first create a thin pool using the `lvcreate` command.
```bash
lvcreate -L 1G -T volume_group_one/mythinpool
```

# OPTIONS
**-debug**=*true*|*false*
  Enable debug logging. Default is false.
**-version**=*true*|*false*
  Print version information and quit. Default is false.

# EXAMPLES
**Volume Creation**
```bash
docker volume create -d lvm --name foobar --opt size=0.2G
```
This will create a lvm volume named foobar of size 208 MB (0.2 GB).
```bash
docker volume create -d lvm --name thin --opt size=0.2G --opt thinpool=mythinpool
```
This will create a thinly-provisioned lvm volume in mythinpool.
```bash
docker volume create -d lvm --name foobar_snapshot --opt snapshot=foobar --opt size=100M
```
This will create a snapshot volume of foobar. For thin snapshots, don't specify a size.

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
        "Name": "foobar",
        "Driver": "lvm",
        "Mountpoint": "/var/lib/docker-lvm-plugin/foobar123"
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



