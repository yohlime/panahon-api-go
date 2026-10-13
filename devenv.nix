{
  pkgs,
  lib,
  config,
  inputs,
  ...
}:
{
  packages = with pkgs; [
    air
    cacert
    coreutils
    (go-migrate.overrideAttrs (oldAttrs: {
      tags = [ "postgres" ];
    }))
    go-mockery
    go-swag
    postgresql_16
    secretspec
    sqlc
  ];

  env.DOCKER_HOST = "unix://$XDG_RUNTIME_DIR/podman/podman.sock";

  languages.go.enable = true;

  services.postgres = {
    enable = true;
    package = pkgs.postgresql_16;
    extensions = extensions: [ extensions.postgis ];
    listen_addresses = "localhost";
    port = 5432;
    initialDatabases = [
      {
        name = "weather";
        user = "weatheradmin";
        pass = "w5r@dm!N";
        initialSQL = ''
          CREATE EXTENSION IF NOT EXISTS postgis;
        '';
      }
    ];
  };

  # `devenv up` starts postgres to readiness, seeds dev data (idempotent), then air.
  tasks."db:seed" = {
    exec = "go run ./cmd/testcli seed all -v";
    after = [ "devenv:processes:postgres" ];
    showOutput = true;
  };

  processes.air = {
    exec = "air";
    after = [ "devenv:processes:postgres" "db:seed" ];
  };

  enterShell = ''
    go version
  '';

  enterTest = ''
    echo "Running tests"
    make test
  '';
}
