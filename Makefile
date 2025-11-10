
watch: air-install
	go tool air

build: templ_generate css
	@export CGO_ENABLED=1
	go build -v -o ./.dist/ ./cmd/aoclb/

templ_generate:
	go tool templ generate ./interbal/web/templates

tailwind-install:
	@if [ ! -f tailwindcss ]; then curl -sL https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64 -o tailwindcss; fi
	
	@chmod +x tailwindcss

air-install:
	go get -tool github.com/air-verse/air@latest


css: tailwind-install
	./tailwindcss --minify -i ./internal/web/styles/tailwind.css -o ./internal/web/assets/css/tailwind.css

