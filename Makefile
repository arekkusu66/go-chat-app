run:
	templ generate && go run .
run-del:
	templ generate && rm -rf sqlite.db && go run .