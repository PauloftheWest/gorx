run:
	LD_LIBRARY_PATH=ext/orx/code/lib/dynamic go run .
orx:
	cd ext/orx ; ./setup.sh
	cd ext/orx/code/build/linux/gmake ; make
