{ pkgs, lib, config, inputs, ... }:

{
    env.ENVIRONMENT = "development";
    env.DATABASE_HOST = "localhost";
    env.DATABASE_PORT = "3307";
    env.DATABASE_USERNAME = "trackme";
    env.DATABASE_PASSWORD = "trackme";
    env.DATABASE_NAME = "tracker";

    languages.go.enable = true;

    services.mysql = {
      enable = true;
      package = pkgs.mysql80;
      initialDatabases = lib.mkDefault [
        { name = "tracker"; }
        { name = "tracker_test"; }
      ];
      ensureUsers = lib.mkDefault [
        {
          name = "trackme";
          password = "trackme";
          ensurePermissions = {
            "tracker.*" = "ALL PRIVILEGES";
            "tracker_test.*" = "ALL PRIVILEGES";
          };
        }
      ];

      settings = {
        mysqld = {
          port = 3307;
          mysqlx = 0; 
          group_concat_max_len = 320000;
          log_bin_trust_function_creators = 1;
          sql_mode = "STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION";
        };
      };      
    };

    scripts.install-mod.exec = ''
        go mod download
    '';

    scripts.build-app.exec = ''
      go build -o tracker cmd/app/main.go
    '';

    scripts.gol.exec = ''
        golangci-lint run
    '';

    scripts.golf.exec = ''
        golangci-lint run --fix
    '';

    scripts.got.exec = ''
        go test ./... 
    '';

    scripts.gotc.exec = ''
        go test $(go list ./... | grep "^gitlab.shopware.com/shopware/6") -coverprofile=coverage-report.out
        go tool cover -html=coverage-report.out
        rm coverage-report.out
    '';

    scripts.waiting_for_db_ready.exec = ''
      MAX_ATTEMPTS=10
      attempts=0
      while [ $attempts -lt $MAX_ATTEMPTS ]; do
        mysql -u "$DATABASE_USERNAME" -p"$DATABASE_PASSWORD" -e "exit" >/dev/null 2>&1
        if [ $? -eq 0 ]; then
          break
        fi
        echo "Waiting for MySQL to be ready..."
        ((attempts++))
        sleep 3
      done
    '';

    scripts.start-app.exec = ''
      
      if [ ! -f ./tracker ]; then
        install-mod && build-app
      fi
      go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

      waiting_for_db_ready

      $GOPATH/bin/migrate -database "mysql://$DATABASE_USERNAME:$DATABASE_PASSWORD@tcp($DATABASE_HOST:$DATABASE_PORT)/$DATABASE_NAME_test" -path db/migrations up
      echo "Migrations applied"

      ./tracker
    '';

    processes.tracker.exec = ''
        start-app
    '';
    
    enterShell = ''
        go install github.com/swaggo/swag/cmd/swag@latest
        alias swag=$GOPATH/bin/swag
        go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
        alias migrate=$GOPATH/bin/migrate
        ./devenv-start.sh  
    '';
}
