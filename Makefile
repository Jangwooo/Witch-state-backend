SHELL = /bin/bash

dev-start:
	docker-compose --env-file .env -f docker-compose.dev.yml up -d --build

prod-start:
	docker pull kjw2262/witchs_lounge_backend:latest
	docker-compose --env-file .env.prod -f docker-compose.prod.yml up -d

manual-push:
	 docker buildx build --platform linux/amd64 --push -f Dockerfile.prod -t kjw2262/witchs_lounge_backend:latest .

update-image:
	docker pull kjw2262/witchs_lounge_backend:latest

swagger:
	swag init -g app/main.go

# Very dangerous!!!
clear-compose:
	docker-compose down --volumes --remove-orphans
	docker rmi $(docker images -q) --force
	docker volume prune --force
	docker network prune --force
	docker container prune --force
	docker image prune --all --force