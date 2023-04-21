.PHONY: start-postgresql
start-postgresql:
	docker run -i --rm \
		--name sqlbench \
		-e POSTGRES_HOST_AUTH_METHOD=trust \
		-e POSTGRES_USER=bench \
		-e POSTGRES_DB=bench \
		postgres:15-alpine -c shared_buffers=1GB -c work_mem=32MB -c effective_cache_size=2GB
