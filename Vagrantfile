# Specify minimum Vagrant version and Vagrant API version
Vagrant.require_version ">= 1.6.0"
VAGRANTFILE_API_VERSION = "2"

# Create box
Vagrant.configure("2") do |config|
  config.vm.define "docker-lvm-plugin-fedora32"
  config.vm.box = "generic/fedora32"
  config.vm.synced_folder ".", "/home/vagrant/go/src/github.com/docker-lvm-plugin"
  config.ssh.extra_args = ["-t", "cd /home/vagrant/go/src/github.com/docker-lvm-plugin; bash --login"]
  config.vm.provider "virtualbox" do |vb|
     vb.name = "docker-lvm-plugin-fedora32"
     vb.cpus = 2
     vb.memory = 2048
  end
  config.vm.provision "shell", inline: <<-SHELL
    dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo
    dnf install -y docker-ce docker-ce-cli containerd.io go-md2man lvm2 cryptsetup
    systemctl start docker

    echo "export GOPATH=/home/vagrant/go" >> /home/vagrant/.bashrc
    echo "export PATH=$PATH:/usr/local/go/bin" >> /home/vagrant/.bashrc
    source /home/vagrant/.bashrc

    # Install golang-1.14.3
    if [ ! -f "/usr/local/go/bin/go" ]; then
      curl -s -L -o go1.14.3.linux-amd64.tar.gz https://dl.google.com/go/go1.14.3.linux-amd64.tar.gz
      sudo tar -C /usr/local -xzf go1.14.3.linux-amd64.tar.gz
      sudo chmod +x /usr/local/go
      rm -f go1.14.3.linux-amd64.tar.gz
    fi

    cd /home/vagrant/go/src/github.com/docker-lvm-plugin
    make
    make install
    systemctl start docker-lvm-plugin

    export TEMP_DIR=$(pwd)/temp
    mkdir -p $TEMP_DIR
    dd if=/dev/zero of=$TEMP_DIR/loopfile bs=1024k count=2000 status=none
    loopdevice=$(losetup --find)
    losetup $loopdevice $TEMP_DIR/loopfile
    pvcreate $loopdevice
    vgcreate vg1 $loopdevice
    lvcreate -L 1G -T vg1/mythinpool
    echo "VOLUME_GROUP=vg1" > /etc/docker/docker-lvm-plugin
  SHELL
end
