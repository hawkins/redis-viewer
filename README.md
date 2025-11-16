# Redis Viewer

A tool to view Redis data in terminal.

![user interface](images/ui.png)

## Install

`go install github.com/hawkins/redis-viewer@latest`


## Usage:

See help: `redis-viewer --help` or press `?` inside the application to see keybindings

Default config file path is `$HOME/.redis-viewer.yaml`

Example config file:

```yaml
addrs:
    - 127.0.0.1:6380
    - 127.0.0.1:6381
    - 127.0.0.1:6382
    - 127.0.0.1:6383
    - 127.0.0.1:6384
    - 127.0.0.1:6385

db:
username:
password:

master_name:
```

## Support:

-   client, sentinel and cluster mode.
-   `string, hash, list, set, zset` key types.

## Note:

In Windows, you should change system encoding to `UTF-8` before run this program.


Built with [bubbletea](https://github.com/charmbracelet/bubbletea).