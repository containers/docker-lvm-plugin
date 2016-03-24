# docker lvm plugin
Docker Volume Driver for lvm volumes

This plugin can be used to create lvm volumes of specified size, which can 
then be bind mounted into the container using `docker run` command.

## Setup

	1) git clone git@github.com:shishir-a412ed/docker-lvm-plugin.git
	2) cd docker-lvm-plugin
	3) make
	4) sudo make install

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
3) Since logical volumes (lv's) are based on a volume group, it is the 
   responsibility of the user (administrator) to provide a volume group name.
   You can choose an existing volume group name by listing volume groups on 
   your system using `vgs` command OR create a new volume group using 
   `vgcreate` command.
   e.g. 
```bash
vgcreate volume_group_one /dev/hda 
```
   where /dev/hda is your partition or whole disk on which physical volumes 
   were created.

4) Add this volume group name in the config file 
```bash
/etc/docker/docker-lvm-plugin
```
## Volume Creation

``` bash
$ docker volume create -d lvm --name foobar --opt size=0.2G
```
This will create a lvm volume named foobar of size 208 MB (0.2 GB).

## Volume List

``` bash
$ docker volume ls
```
This will list volumes created by all docker drivers including the default driver (local).

## Volume Inspect

``` bash
$ docker volume inspect foobar
```
This will inspect foobar and return a JSON.
```bash
[
    {
        "Name": "foobar",
        "Driver": "lvm",
        "Mountpoint": "/var/lib/docker-lvm-plugin/foobar123"
    }
]
```

## Volume Removal
```bash
$ docker volume rm foobar
```
This will remove lvm volume foobar.

## Bind Mount lvm volume inside the container

```bash
$ docker run -it -v foobar:/home fedora /bin/bash
```
This will bind mount the logical volume `foobar` into the home directory of the container.

## License
GNU GPL





