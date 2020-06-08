# SIG CLI KEP FAQ

## Why not as a kubectl plugin instead of compiled in?

- The kubectl plugin mechanism does not provide a solution for distribution.  Because the functionality is intended as
 the project's solution to issues within kubectl, we want it to be available to users of kubectl without additional
 steps.  Having users manually download only Kustomize as a plugin might be ok, but it won't scale as a good approach
 as the set of commands grows.
- The effort to build and test the tool for all targets, develop a release process, etc. would be much higher for SIG
  CLI, also, and it would exacerbate kubectl's version-skew challenges.
- It will not support integration at more than a surface level - such as into the resource builder
  (which does not offer a plugin mechanism).
    - It was previously decided we didn't want to add a plugin mechanism to the resource builder.
      This could be reconsidered, but would need to think through it more and figure out how to address
      previously brought up issues.  There may be other issues not listed here as well.
      - https://github.com/kubernetes/kubernetes/issues/13241
      - https://github.com/kubernetes/kubernetes/pull/14993
      - https://github.com/kubernetes/kubernetes/pull/14918
- There is a risk that publishing each command as a separately built binary could cause the aggregate download
  size of the toolset to balloon.  The kubectl binary is *52M* and the kustomize binary is *31M*.  (extrapolate to
  30+ commands x 30MB).  Before going down this route, we should consider how to we might want to design a solution
  and the tradeoffs.

