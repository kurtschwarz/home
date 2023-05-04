# Home Assistant

## Setting up Lutron Caseta

1. Install `tron` (see [https://github.com/paulrosania/tron]())
2. Configure `~/.tronrc` by running `echo "host=10.0.14.82" > ~/.tronrc`
3. Run `tron pair` to generate the required certificates
4. Load the certificates into `pulumi` using the `config set` command
   1. `cat ~/.config/tron/certs/ca.crt | pulumi config set --path 'home-assistant:secrets.LUTRON_CASETA_CA_CERT' --secret`
   2. `cat ~/.config/tron/certs/client.crt | pulumi config set --path 'home-assistant:secrets.LUTRON_CASETA_CLIENT_CERT' --secret`
   3. `cat ~/.config/tron/certs/client.key | pulumi config set --path 'home-assistant:secrets.LUTRON_CASETA_CLIENT_KEY' --secret`
