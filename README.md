# less-tree

A tool to batch your server-side LESS compilations.

## Introduction

Writing LESS is great, but there's always the question of how to convert .less files into .css files so they're ready to serve on the web. The "official" way is to run a command-line program, `lessc` and pipe it to a file:

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

## Other options

Typing `less-tree -help` yields this output:

```text
Usage of less-tree:
  -css-min="": Path to a CSS minifier which takes an input file and spits out minified CSS in stdout
  -max-jobs=10: Maximum amount of jobs to run at once
  -path="lessc": Path to the lessc executable
  -v=false: Whether or not to show LESS errors
```

* The **css-min** flag can point to a different CSS minifier if you want, rather than `lessc -x`. You might want to check out [this node.js cssmin port][1].
* The **max-jobs** flag can set a maximum amount of jobs (compilations) to run at once. I'd recommend leaving this at the default, but you can increase or decrease as you want, your mileage may vary.
* Set **path** if `less-tree` can't access the `lessc` executable from npm.

## Future development

* I'd like to integrate a full, compiled LESS compiler, to increase performance and remove the dependency on node.js.

## License

```none
The MIT License (MIT)
Copyright (C) 2013 by Jimmy Sawczuk

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

  [1]: https://github.com/jbleuzen/node-cssmin