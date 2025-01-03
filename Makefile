
export SHELL := /bin/sh

LOCAL_BIN := $(CURDIR)/.bin
OAPI_CODEGEN_VERSION := v2.2.0
OAPI_CODEGEN_BIN=$(LOCAL_BIN)/oapi-codegen

LATEST_GO := $$(cd latestgo && go run main.go)

BACKEND_URL := http://sandbox_dev/run
#BACKEND_URL := http://host.docker.internal:8080/run


#cd api && $(OAPI_CODEGEN_BIN) -package api -generate client api.yml > ./gen/client.gen.go
#cd api && $(OAPI_CODEGEN_BIN) -package api -generate chi-server,spec api.yml > ./gen/server.gen.go

contracts-generate:
	#which $(OAPI_CODEGEN_BIN)
	# go types
	cd api &&  $(OAPI_CODEGEN_BIN) -package api -generate types api.yml > ./gen/api.gen.go
	cd api &&  $(OAPI_CODEGEN_BIN) -package api -generate types sandbox.yml > ./gen/sandbox.gen.go
	cd api &&  $(OAPI_CODEGEN_BIN) -package api -generate types sandbox.yml > ../sandbox/api/gen/sandbox.gen.go


.PHONY: install-oapi-codegen
install-oapi-codegen:
	$(call fn_install_goutil,github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen,$(OAPI_CODEGEN_VERSION),$(OAPI_CODEGEN_BIN),"-tags=''")

# fn_install_goutil устанавливает бинарь из гошного пакета.
# Параметры:
# 1 - uri пакета для сборки бинаря;
# 2 - версия пакета вида vX.Y.Z или latest;
# 3 - полный путь для установки бинаря.
# 4 - опциональные build флаги
# Работает не через go install, чтобы иметь возможность использовать разные версии в разных модулях и не добавлять пакет в зависимости текущего модуля.
# Проверяет наличие бинаря, создаёт временную директорию, инициализирует в ней временный модуль, в котором вызывает установку и сборку бинаря.
define fn_install_goutil
	@[ ! -f $(3)@$(2) ] \
		|| exit 0 \
		&& echo "Install $(1) ..." \
		&& tmp=$$(mktemp -d) \
		&& cd $$tmp \
		&& echo "Module: $(1)" \
		&& echo "Version: $(2)" \
		&& echo "Binary: $(3)" \
		&& echo "Temp: $$tmp" \
		&& go mod init temp && go get -d $(1)@$(2) && go build $(4) -o $(3)@$(2) $(1) \
		&& ln -sf $(3)@$(2) $(3) \
		&& rm -rf $$tmp \
		&& echo "$(3) installed!" \
		&& echo "********************************"
endef

rebuild:
	COMPOSE_PROFILES=prebuild,build docker compose down --rmi local
	rm -rf ./var
	COMPOSE_PROFILES=prebuild docker compose up
	COMPOSE_PROFILES=build docker compose up