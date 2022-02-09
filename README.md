This is the sample code for a post on the Khan Academy Engineering Blog
describing how we use strongly typed context.
TODO: Insert link to blog post.

There are 5 examples, each of which is described in the blog post.  In each
case the file `thing.go` contains a function `DoTheThing` which does the same
things.  They vary in how they access global & request specific elements.

1. Globals
2. Parameters
3. Context, with unsafe casting
4. Context but safely
5. Strongly typed context

The blog post explains the pros and cons, and explains why we use strongly
typed contexts at Khan Academy.

There's also a `main` that calls `DoTheThing`.  Comparing the differences
between the examples allows you to see how easy it is to write functions and
call functions using the various techniques.

Each example also has a file `mock.go` which has stub implementations of the
functions called in `DoTheThing`.  The last example also has `contexts.go`
which defines the interface types used in the example.

Finally there's a linter that can detect when a function declaration requires an
interface type that is not used in the code.  We use this linter to ensure every
function declares the minimal interface possible, which ensures the interface
list is an accurate list of dependencies.  If you run it against example 5 it
passes.  In `mocks.go` there are lines like `_ = ctx.Request()` that exist to
satisfy the linter.  If those lines are removed the linter will fail.

Khan Academy is providing the linter AS-IS as a proof of concept.  We are not
taking pull-requests and can't help with making it work for you, but you're
welcome to adapt it for linting your own code.


We use strongly typed contexts within Khan Academy.  If you like the idea and
are excited to use them at work, [we're hiring](https://www.khanacademy.org/careers).

