all: bin/redis2http

PLATFORM=local

.PHONY: bin/redis2http
bin/redis2http:
	@docker build . --target bin \
	--output bin/ \
	--platform ${PLATFORM}
