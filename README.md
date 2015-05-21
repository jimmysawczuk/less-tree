# less-tree

A tool to batch your server-side [LESS][3] compilations.

## Introduction

[Writing LESS is great][2], but there's always the question of how to convert .less files into .css files so they're ready to serve on the web. The "official" way is to run a command-line program, `lessc` and pipe it to a file:

```bash
lessc path/to/less/sheet.less > path/to/css/sheet.css
```

You have to run this again if you want the minified version too, with a `-x` switch.

I used to write Makefiles in which I'd set up rules for each LESS sheet and explicitly say where each file should be compiled. It worked, but it took time and effort to set up. I also had written a Bash script that looked for all valid *.less files and tried to compile them to similar paths, and this worked reasonably well, but it was kind of slow.

## The solution

**less-tree** is written in Go, and it is able to run `lessc` on many files at once to increase the efficiency of compilation and reduce the amount of time it takes to do a full compile of all your files. Basically, the program expects you to have your directories set up like this:

```text
www/
|-- css/
|-- less/
   |-- sub-dir/
      |-- style-sub.less
   |-- style-1.less
   |-- style-2.less
```

And all you have to do is run:

```bash
less-tree www
```

This will compile every *.less file in the the `less` subdirectory of `www`, and put the resulting stylesheets (both standard and minified) in `css`, keeping the directory structures intact. It's also able to multithread and compile more than one LESS stylesheet at a time, which should help things work way faster.

## Includes

less-tree will skip any files or directories prefixed with a `_`. If you have LESS files that are only used as includes and don't produce independent CSS output, prefix them with a `_`.

## Requirements

* `lessc` installed as a command-line program via npm. You can get more details [here][3], or you can just run `npm install -g less`.
* `cssmin` (optional) is required if you want to minify your outputted CSS, you can install it via `npm install -g cssmin`.

## Other options

Typing `less-tree -help` yields this output:

```text
Usage: less-tree [options] <dir> <another-dir>...
  -cssmin-path="": Path to cssmin (or an executable which takes an input file as an argument and spits out minified CSS in stdout)
  -lessc-path="lessc": Path to the lessc executable
  -max-jobs=10: Maximum amount of jobs to run at once
  -min=false: Automatically minify outputted css files
  -v=false: Whether or not to show LESS errors
```

* `lessc -x` is deprecated so LESS is no longer automatically minified. Installing `cssmin` is pretty easy and you can use `cssmin` automatically via `-min -cssmin-path="cssmin"`.
* The `cssmin-path` flag can point to a different CSS minifier if you want, rather than `cssmin`.
* The **max-jobs** flag can set a maximum amount of jobs (compilations) to run at once. I'd recommend leaving this at the default, but you can increase or decrease as you want, your mileage may vary.
* Set `-lessc-path=/full/path/to/lessc` if `less-tree` can't access the `lessc` executable, or `lessc` isn't in your PATH.

## License

```text
The MIT License (MIT)
Copyright (C) 2013-2015 by Jimmy Sawczuk

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
```

  [2]: http://www.jimmysawczuk.com/2011/11/less-is-more.html
  [3]: http://www.lesscss.org
