{ ... }:

{
  languages = {
    go = {
      enable = true;
      version = "1.26.0";
    };
  };

  git-hooks = {
    enable = true;
    hooks = {
      convco = {
        enable = true;
      };
    };
  };
}
