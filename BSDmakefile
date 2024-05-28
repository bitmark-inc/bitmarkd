# BSDmakefile
# forward flags, vars, and targets to gmake

.export

.if "${.TARGETS}" == ""

.PHONY: all
all:
	gmake ${MAKEFLAGS}

.else

.PHONY: ${.TARGETS}
${.TARGETS}: _internal_

.PHONY: _internal_
_internal_:
	gmake ${MAKEFLAGS} ${.TARGETS}

.endif
