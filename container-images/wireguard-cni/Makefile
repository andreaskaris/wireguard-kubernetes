build-fedora:
	make -C ../../controller build
	cp ../../controller/bin/debug bin/debug
	docker build --file Dockerfile.fedora -t wireguard-cni . 
