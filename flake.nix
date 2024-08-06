{
  description = "KRM functions to manage SealedSecrets";

  inputs = {
    devenv-root = {
      url = "file+file:///dev/null";
      flake = false;
    };
    nixpkgs.url = "github:cachix/devenv-nixpkgs/rolling";
    devenv.url = "github:cachix/devenv";
    nix2container.url = "github:nlewo/nix2container";
    nix2container.inputs.nixpkgs.follows = "nixpkgs";
    mk-shell-bin.url = "github:rrbutani/nix-mk-shell-bin";

    gomod2nix.url = "github:nix-community/gomod2nix";
    gomod2nix.inputs.nixpkgs.follows = "nixpkgs";

    kpt.url = "github:jashandeep-sohi/kpt";
  };

  nixConfig = {
    extra-trusted-public-keys = "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=";
    extra-substituters = "https://devenv.cachix.org";
  };

  outputs = inputs@{ flake-parts, devenv-root, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devenv.flakeModule
      ];
      systems = [ "x86_64-linux" "i686-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];

      perSystem = { config, self', inputs', pkgs, system, ... }:
      let
        version = with inputs; "${self.shortRev or self.dirtyShortRev or "unknown"}";
        homepage = "https://github.com/jashandeep-sohi/krm-fn-sealedsecrets";
        buildGoCmd = { pname, cmd }: inputs'.gomod2nix.legacyPackages.buildGoApplication {
          inherit pname version;
          src = pkgs.lib.cleanSource ./.;
          modules = ./gomod2nix.toml;
          subPackages = [ "cmd/${cmd}" ];
          postInstall = ''
            mv $out/bin/${cmd} $out/bin/${pname}
          '';
          ldflags = [
            "-s" "-w"
            "-X github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/version.Name=${version}"
            "-X github.com/jashandeep-sohi/krm-fn-sealedsecrets/pkg/version.URL=${homepage}"
          ];
          meta = {
            inherit homepage;
          };
        };
        user = "nobody";
        group = "nobody";
        uid = "1000";
        gid = "1000";
        makeImageUser = pkgs.runCommand "mkUser" { } ''
            mkdir -p $out/etc/pam.d
            echo "${user}:x:${uid}:${gid}::" > $out/etc/passwd
            echo "${user}:!x:::::::" > $out/etc/shadow
            echo "${group}:x:${gid}:" > $out/etc/group
            echo "${group}:x::" > $out/etc/gshadow
        '';
        buildImage = { name, tag, package }: with inputs'.nix2container.packages; nix2container.buildImage {
          inherit name tag;
          copyToRoot = [ makeImageUser ];
          perms = [
            { path = makeImageUser; regex = ".*"; mode = "0664"; uname = "nobody"; gname = "nobody"; }
          ];
          config = {
            User = user;
            Entrypoint = [
                "${package}/bin/${package.pname}"
            ];
          };
        };
      in {
        # Per-system attributes can be defined here. The self' and inputs'
        # module parameters provide easy access to attributes of the same
        # system.

        packages.default = config.packages.seal;
        packages.seal = buildGoCmd { pname = "krm-fn-sealedsecrets-seal"; cmd = "seal"; };
        packages.unseal = buildGoCmd { pname = "krm-fn-sealedsecrets-unseal"; cmd = "unseal"; };

        packages.sealContainer = buildImage { name = "ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/seal"; tag = "latest";  package = config.packages.seal; };
        packages.unsealContainer = buildImage { name = "ghcr.io/jashandeep-sohi/krm-fn-sealedsecrets/unseal"; tag = "latest";  package = config.packages.unseal; };

        packages.gomod2nix = inputs'.gomod2nix.packages.default;

        devenv.shells.default = {
          devenv.root =
            let
              devenvRootFileContent = builtins.readFile devenv-root.outPath;
            in
            pkgs.lib.mkIf (devenvRootFileContent != "") devenvRootFileContent;

          # https://devenv.sh/reference/options/
          packages = [
            pkgs.devenv
            pkgs.kustomize
            pkgs.kubeseal
            inputs'.kpt.packages.default
            config.packages.gomod2nix
          ];

          enterShell = ''
            export SHELL=${pkgs.bashInteractive}/bin/bash
          '';

          languages.nix.enable = true;
          languages.go.enable = true;

          pre-commit.hooks = {
            gofmt.enable = true;
            govet.enable = true;
            golangci-lint.enable = true;

            gomod2nix = {
              enable = true;
              name = "gomod2nix";
              entry = "gomod2nix";
              files = "go\\.(mod|sum)";
              pass_filenames = false;
            };
          };

        };

      };
      flake = {
        # The usual flake attributes can be defined here, including system-
        # agnostic ones like nixosModule and system-enumerating ones, although
        # those are more easily expressed in perSystem.

      };
    };
}
