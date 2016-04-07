# less-tree

[![GoDoc](https://godoc.org/github.com/jimmysawczuk/less-tree?status.svg)][godoc-link] [![go report card](https://goreportcard.com/badge/jimmysawczuk/less-tree)][goreportcard-link]

A tool to batch your server-side [LESS][lesscss] compilations.

**less-tree** runs `lessc` on many LESS files at once, increasing the throughput and decreasing the amount of time it takes to do a full compile of all your LESS files.

less-tree assumes your public folder looks like this:

```text
public/
|-- css/
|-- less/
   |-- sub-dir/
      |-- style-sub.less
   |-- _include.less
   |-- style-1.less
   |-- style-2.less
```

And all you have to do is run:

```bash
less-tree www
```

This will compile every *.less file in the the `less` subdirectory of `www`, and put the resulting stylesheets in the `css` subdirectory, keeping the directory structures intact. Your public directory will look like this:

```text
public/
|-- css/
   |-- sub-dir/
      |-- style-sub.css
   |-- style-1.css
   |-- style-2.css
|-- less/
   |-- sub-dir/
      |-- style-sub.less
   |-- _include.less
   |-- style-1.less
   |-- style-2.less
```

## Other features:

* **Includes:** less-tree treats any file or directory prefixed with a `_` as a non-output LESS file, meaning it assumes it's only used as an include and won't run `lessc` on those files independently.
* **Minification:** less-tree can optionally minify your CSS as well, using `cssmin`. The minified versions will be stored parallel to the non-minified versions. Simply pass `-min -cssmin-path="/path/to/cssmin"`.
* **Intelligent caching:** by default, less-tree will only compile LESS files with changes or LESS files with imports that have changed (you can force a recompile of everything using `-f`). less-tree keeps track of what's changed in a JSON file in `<public_dir>/css/.less-tree-cache`. There is probably not much inherently risky in keeping it accessible, but if you want to block access to it, an `.htaccess` in `<public_dir>/css` with the following should do the trick:

```plain
<Files ".less-tree-cache">
  Order Allow,Deny
  Deny from all
</Files>
```

## Requirements

less-tree doesn't compile anything on its own (yet), so you'll need to be able to install a couple of [npm nodules][npm]

* `lessc` installed as a command-line program via npm. You can get more details [here][lesscss], or you can just run `npm install -g less`.
* `cssmin` (optional) is required if you want to minify your outputted CSS, you can install it via `npm install -g cssmin`.

## Help

Type `less-tree -help` to see a full command reference.

## License

less-tree is released under [the MIT license][license].

  [godoc-link]: https://godoc.org/github.com/jimmysawczuk/less-tree
  [goreportcard-link]: https://goreportcard.com/report/github.com/jimmysawczuk/less-tree
  [lesscss]: http://www.lesscss.org
  [npm]: http://www.npmjs.com
  [license]: https://github.com/jimmysawczuk/less-tree/blob/master/LICENSE
