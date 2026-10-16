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

  # testcontainers needs the podman socket. Set DOCKER_HOST in the shell so
  # $XDG_RUNTIME_DIR is expanded (a Nix string would keep it literal).
  enterShell = ''
    export DOCKER_HOST="unix://$XDG_RUNTIME_DIR/podman/podman.sock"
    go version
  '';

  enterTest = ''
    export DOCKER_HOST="unix://$XDG_RUNTIME_DIR/podman/podman.sock"
    echo "Running tests"
    make test
  '';
}
