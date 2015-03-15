Bort is an IRC bot with plugin capability, written in the Go programming
language.

The bot consists of the bort command, which handles the IRC connection, and the
bortplug command, which handles plugins.  The bortplug command can be stopped,
recompiled with different or reconfigured plugins, and restarted while the bort
command stays commected to the IRC server.

Plugins may implement commands, respond to matched text, or push messages
asynchronously.  Plugins are compiled into the bortplug command.  To enable a
plugin, add 'import _ "plugin_import_path"' to cmd/bortplug/plugins.go.

Bort looks for a JSON configuration file in ~/.config/bort/bort.conf, which can
be overridden with a command line parameter.  Bort prioritizes command line
parameter values, followed by configuration file, and finally, default values.
Plugins have access to the configuration file data, and may look for values of
an appropriate key.

See the [documentation](https://godoc.org/github.com/ianremmler/bort) for more
information.
