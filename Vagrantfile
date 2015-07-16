Vagrant.configure(2) do |config|

  config.vm.box = "ubuntu/trusty64"
  config.vm.provider "lxc" do |v, override|
    override.vm.box = "fgrehm/trusty64-lxc"
  end

  # Expose ports
  config.vm.network "forwarded_port", host: 9900, guest: 9900

  # Inject user files
  ['~/.vimrc'].each do |file|
    config.vm.provision :file, source: file, destination: file if File.exist?(File.expand_path(file))
  end

  # Inject user dirs
  ['.vim'].each do |dir|
    config.vm.synced_folder '~/' + dir, '/home/vagrant/' + dir if File.exist?(File.expand_path('~/' + dir))
  end

  # Install dependencies and build
  config.vm.provision :shell, inline: "apt-get update"
  config.vm.provision :shell, inline: "apt-get install -y build-essential git mercurial"
  config.vm.provision :shell, inline: "cd /vagrant && make test", privileged: false

  # Make editors happy ;)
  config.vm.provision :shell, inline: "echo 'export GOPATH=/vagrant/.build/gopath' >> ~/.profile", privileged: false
  config.vm.provision :shell, inline: "echo 'export PATH=$PATH:/vagrant/.build/go1.4.2/bin' >> ~/.profile", privileged: false

end
