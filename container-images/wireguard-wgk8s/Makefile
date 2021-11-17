build-fedora:
	make -C ../../controller build
	cp ../../controller/bin/wgk8s bin/wgk8s
	docker build --file Dockerfile.fedora -t wireguard-wgk8s . 
