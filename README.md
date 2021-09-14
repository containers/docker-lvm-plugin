# docker lvm plugin
[![CircleCI](https://circleci.com/gh/containers/docker-lvm-plugin/tree/master.svg?style=shield&circle-token=af715d8feae4c98aa240cfae4985a2bb0ee017bb)](https://circleci.com/gh/containers/docker-lvm-plugin/tree/master)
[![License: GNU LGPL v3.0](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://github.com/containers/docker-lvm-plugin/blob/master/LICENSE)
[![Release](https://img.shields.io/badge/version-1.0-blue)](https://github.com/containers/docker-lvm-plugin/releases/tag/v1.0)

Docker Volume Driver for lvm volumes

This plugin can be used to create lvm volumes of specified size, which can
then be bind mounted into the container using `docker run` command.

## Setup
### Using Docker
    docker plugin install --alias lvm containers/docker-lvm-plugin/docker-lvm-plugin VOLUME_GROUP=vg0

### Manual
    1) git clone git@github.com:projectatomic/docker-lvm-plugin.git (You can also use HTTPS to clone: git clone https://github.com/projectatomic/docker-lvm-plugin.git)
    2) cd docker-lvm-plugin
    3) export GO111MODULE=on
    4) make
    5) sudo make install

## Wanna try it out!? (Vagrant)

From the `$root` directory of the project.

```
$ vagrant up
```

Once the VM is up and running (`vagrant global-status` will list you all the vagrant VMs running on your system), you can `ssh` into the VM by running `vagrant ssh docker-lvm-plugin-fedora33`.

## Screencast
[![asciicast](https://asciinema.org/a/248482.svg)](https://asciinema.org/a/248482)

## Usage

1) Start the docker daemon before starting the docker-lvm-plugin daemon.
   You can start docker daemon using command:
```bash
sudo systemctl start docker
```
2) Once docker daemon is up and running, you can start docker-lvm-plugin daemon
   using command:
```bash
sudo systemctl start docker-lvm-plugin
```
NOTE: docker-lvm-plugin daemon is on-demand socket activated. Running `docker volume ls` command
will automatically start the daemon.

3) Since logical volumes (lv's) are based on a volume group, it is the
   responsibility of the user (administrator) to provide a volume group name.
   You can choose an existing volume group name by listing volume groups on
   your system using `vgs` command OR create a new volume group using
   `vgcreate` command.
   e.g.
```bash
vgcreate vg0 /dev/hda
```
   where /dev/hda is your partition or whole disk on which physical volumes
   were created.

4) Add this volume group name in the config file
```bash
/etc/docker/docker-lvm-plugin
```

5) The docker-lvm-plugin also supports the creation of thinly-provisioned volumes. To create a thinly-provisioned volume, a user (administrator) must first create a thin pool using the `lvcreate` command.
```bash
lvcreate -L 10G -T vg0/mythinpool
```
This will create a thinpool named `mythinpool` of size 10G under volume group `vg0`.
NOTE: thinpools are special kind of logical volumes carved out of the volume group.
Hence in the above example, to create the thinpool `mythinpool` you must have atleast 10G of freespace in volume group `vg0`.

6) The docker-lvm-plugin allows you to create volumes using an optional volume group, which you can pass using `--opt vg` in `docker volume create` command. However, this is **not recommended** and user (administrator) should stick to the default volume group specified in /etc/docker/docker-lvm-plugin config file.

   If a user still chooses to create a volume using an optional volume group
   e.g `--opt vg=vg1`, user **must** pass `--opt vg=vg1` when creating any derivative volumes
   based off this original volume. E.g

   * Any snapshot volumes which are created off a volume that was created using the optional volume group.
   * Any thin volumes which are created off a thin pool that was created using an optional volume group.

## Volume Creation
`docker volume create` command supports the creation of regular lvm volumes, thin volumes, snapshots of regular and thin volumes.

Usage: docker volume create [OPTIONS]
```bash
-d, --driver    string    Specify volume driver name (default "local")
--label         list      Set metadata for a volume (default [])
--name          string    Specify volume name
-o, --opt       map       Set driver specific options (default map[])
```
Following options can be passed using `-o` or `--opt`
```bash
--opt size
--opt thinpool
--opt snapshot
--opt keyfile
--opt vg
```
Please see examples below on how to use these options.

## Examples
```bash
$ docker volume create -d lvm --opt size=0.2G --name foobar
```
This will create a lvm volume named `foobar` of size 208 MB (0.2 GB) in the
volume group vg0.
```bash
$ docker volume create -d lvm --opt size=0.2G --opt vg=vg1 --name foobar
```
This will create a lvm volume named `foobar` of size 208 MB (0.2 GB) in the
volume group vg1.
```bash
docker volume create -d lvm --opt size=0.2G --opt thinpool=mythinpool --name thin_vol
```
This will create a thinly-provisioned lvm volume named `thin_vol` in mythinpool.
```bash
docker volume create -d lvm --opt snapshot=foobar --opt size=100M --name foobar_snapshot
```
This will create a snapshot volume of `foobar` named `foobar_snapshot`. For thin snapshots, use the same command above but don't specify a size.
```bash
docker volume create -d lvm --opt size=0.2G --opt keyfile=/root/key.bin --name crypt_vol
```
This will create a LUKS encrypted lvm volume named `crypt_vol` with the contents of `/root/key.bin` as a binary passphrase. Snapshots of encrypted volumes use the same key file. The key file must be present when the volume is created, and when it is mounted to a container.

## Volume List
Use `docker volume ls --help` for more information.

``` bash
$ docker volume ls
```
This will list volumes created by all docker drivers including the default driver (local).

## Volume Inspect
Use `docker volume inspect --help` for more information.

``` bash
$ docker volume inspect foobar
```
This will inspect `foobar` and return a JSON.
```bash
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
```

## Volume Removal
Use `docker volume rm --help` for more information.
```bash
$ docker volume rm foobar
```
This will remove lvm volume `foobar`.

## Bind Mount lvm volume inside the container

```bash
$ docker run -it -v foobar:/home fedora /bin/bash
```
This will bind mount the logical volume `foobar` into the home directory of the container.

## Tests
```
sudo make test
```
**NOTE**: These are destructive tests and can leave the system in a changed state.<br/>
It is highly recommended to run these tests either as part of a CI/CD system or on
a immutable infrastructure e.g VMs.

## Currently supported environments.
Fedora, RHEL, Centos, Ubuntu (>= 16.04)

## License
GNU GPL
