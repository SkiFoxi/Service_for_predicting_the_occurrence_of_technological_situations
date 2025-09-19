createdbserv:
	docker exec -it postgresW createdb --username=root --owner=root base_service

dropdbserv:
	docker exec -it postgresW dropdb base_service

.PHONY: createdb dropdb