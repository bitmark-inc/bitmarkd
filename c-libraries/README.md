# build static libraries

Just run **make** in here before  running go install

Note that the Makefile must produce for all libraries in this
directory, the following items:

1. libX/include/*.h
2. libX/libX.a

The pkg-config wrapper script will detect if there is already a system
version of the library, if not it will create a local .pc file to
arrange static linking.  The preference is to use the system libraries.

Notes:

1. the subdirectories _must_ be named lib* as the same name as the
   lib*.a file produced by make for the pkg-config to detect them.
2. the Makefile must be modified to process all libraries and must
   ensure that the two conditions above are met.  i.e. copy/move fiels
   as necessary.
3. the pkg-config wrapper adapts to whatever lib* directories are
   here; no changes needed to add/remove and library.
