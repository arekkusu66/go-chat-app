run:
	sqlc generate && templ generate && go run .
run-migrate:
	sqlc generate && templ generate && go run . migrate
generate:
	sqlc generate && templ generate