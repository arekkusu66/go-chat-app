run: generate
	go run .
run-migrate: generate
	go run . migrate
generate:
	sqlc generate && templ generate