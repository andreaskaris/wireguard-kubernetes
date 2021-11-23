build-fedora:
	make -C ../../controller build
	cp ../../controller/bin/wgcni bin/wgcni
	docker build --file Dockerfile.fedora -t wireguard-cni . 
