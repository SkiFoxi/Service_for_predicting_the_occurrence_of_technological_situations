createdbserv:
	docker exec -it postgresW createdb --username=root --owner=root base_service

dropdbserv:
	docker exec -it postgresW dropdb base_service

upmigration: 
	migrate -path db/migration -database postgresql://root:secret@localhost:5434/base_service?sslmode=disable -verbose up

downmigration:
	migrate -path db/migration -database postgresql://root:secret@localhost:5434/base_service?sslmode=disable -verbose down
	
.PHONY: createdb dropdb