# Makefile
#
# this must compile each of the lib* directories to produce:
# a) libX/include/*.h
# b) libX/libX.a

.PHONY: all
all: libargon2

.PHONY: clean
clean:
	cd libargon2 && [ -f Makefile ] && ${MAKE} clean || true

.PHONY: libargon2
libargon2:
	cd libargon2 && ${MAKE} all
