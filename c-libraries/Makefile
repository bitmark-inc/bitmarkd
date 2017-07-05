# Makefile
#
# this must compile each of the lib* directories to produce:
# a) libX/include/*.h
# b) libX/libX.a

.PHONY: all
all: libargon2 libucl

.PHONY: clean
clean:
	cd libargon2 && [ -f Makefile ] && ${MAKE} clean || true
	cd libucl && [ -f Makefile ] && ${MAKE} clean || true


.PHONY: libargon2
libargon2:
	cd libargon2 && ${MAKE} all

.PHONY: libucl
libucl:
	cd libucl && \
	  ./autogen.sh && \
	  ./configure --disable-debug --disable-dependency-tracking --disable-silent-rules --prefix=/usr && \
	  ${MAKE} && \
	  cp -p src/.libs/libucl.a .
