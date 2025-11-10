
aoclb: templ_generate
	@export CGO_ENABLED=1
	go build -v -o ./.dist/ ./cmd/aoclb/
	@.dist/aoclb

templ_generate:
	go tool templ generate ./interbal/web/templates

tailwind-install:
	@if [ ! -f tailwindcss ]; then curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 -o tailwindcss; fi
	
	@chmod +x tailwindcss

css: tailwind-install
	./tailwindcss --minify -i ./internal/web/styles/tailwind.css -o ./internal/web/assets/css/tailwind.css
