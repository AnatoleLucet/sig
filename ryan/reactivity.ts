//#region src/index.ts
let ReactiveFlags = /* @__PURE__ */ (function (ReactiveFlags$1) {
  ReactiveFlags$1[(ReactiveFlags$1["None"] = 0)] = "None";
  ReactiveFlags$1[(ReactiveFlags$1["Check"] = 1)] = "Check";
  ReactiveFlags$1[(ReactiveFlags$1["Dirty"] = 2)] = "Dirty";
  ReactiveFlags$1[(ReactiveFlags$1["RecomputingDeps"] = 4)] = "RecomputingDeps";
  ReactiveFlags$1[(ReactiveFlags$1["InHeap"] = 8)] = "InHeap";
  ReactiveFlags$1[(ReactiveFlags$1["InHeapHeight"] = 16)] = "InHeapHeight";
  ReactiveFlags$1[(ReactiveFlags$1["Zombie"] = 32)] = "Zombie";
  return ReactiveFlags$1;
})({});
let AsyncFlags = /* @__PURE__ */ (function (AsyncFlags$1) {
  AsyncFlags$1[(AsyncFlags$1["None"] = 0)] = "None";
  AsyncFlags$1[(AsyncFlags$1["Pending"] = 1)] = "Pending";
  AsyncFlags$1[(AsyncFlags$1["Error"] = 2)] = "Error";
  AsyncFlags$1[(AsyncFlags$1["Uninitialized"] = 4)] = "Uninitialized";
  return AsyncFlags$1;
})({});
var NotReadyError = class extends Error {
  constructor(cause) {
    super();
    this.cause = cause;
  }
};
let markedHeap = false;
let stale = false;
let context = null;
let clock = 0;
let transition = null;
let asyncNodes = [];
let pendingNodes = [];
const dirty = {
  heap: new Array(2e3).fill(void 0),
  min: 0,
  max: 0,
};
const pending = {
  heap: new Array(2e3).fill(void 0),
  min: 0,
  max: 0,
};
const NOT_PENDING = {};
function increaseHeapSize(n) {
  if (n > dirty.heap.length) dirty.heap.length = n;
}
function actualInsertIntoHeap(n, heap) {
  const height = n.height;
  const heapAtHeight = heap.heap[height];
  if (heapAtHeight === void 0) heap.heap[height] = n;
  else {
    const tail = heapAtHeight.prevHeap;
    tail.nextHeap = n;
    n.prevHeap = tail;
    heapAtHeight.prevHeap = n;
  }
  if (height > heap.max) heap.max = height;
}
function insertIntoHeap(n, heap) {
  let flags = n.flags;
  if (flags & (ReactiveFlags.InHeap | ReactiveFlags.RecomputingDeps)) return;
  if (flags & ReactiveFlags.Check)
    n.flags =
      (flags & ~(ReactiveFlags.Check | ReactiveFlags.Dirty)) |
      ReactiveFlags.Dirty |
      ReactiveFlags.InHeap;
  else n.flags = flags | ReactiveFlags.InHeap;
  if (!(flags & ReactiveFlags.InHeapHeight)) actualInsertIntoHeap(n, heap);
}
function insertIntoHeapHeight(n, heap) {
  let flags = n.flags;
  if (
    flags &
    (ReactiveFlags.InHeap |
      ReactiveFlags.RecomputingDeps |
      ReactiveFlags.InHeapHeight)
  )
    return;
  n.flags = flags | ReactiveFlags.InHeapHeight;
  actualInsertIntoHeap(n, heap);
}
function deleteFromHeap(n, heap) {
  const flags = n.flags;
  if (!(flags & (ReactiveFlags.InHeap | ReactiveFlags.InHeapHeight))) return;
  n.flags = flags & ~(ReactiveFlags.InHeap | ReactiveFlags.InHeapHeight);
  const height = n.height;
  if (n.prevHeap === n) heap.heap[height] = void 0;
  else {
    const next = n.nextHeap;
    const dhh = heap.heap[height];
    const end = next ?? dhh;
    if (n === dhh) heap.heap[height] = next;
    else n.prevHeap.nextHeap = next;
    end.prevHeap = n.prevHeap;
  }
  n.prevHeap = n;
  n.nextHeap = void 0;
}
function computed(fn, initialValue) {
  const self = {
    disposal: null,
    fn,
    value: initialValue,
    height: 0,
    child: null,
    nextHeap: void 0,
    prevHeap: null,
    deps: null,
    depsTail: null,
    subs: null,
    subsTail: null,
    parent: context,
    nextSibling: null,
    firstChild: null,
    flags: ReactiveFlags.None,
    asyncFlags: AsyncFlags.Uninitialized,
    time: clock,
    pendingValue: NOT_PENDING,
    pendingDisposal: null,
    pendingFirstChild: null,
  };
  self.prevHeap = self;
  if (context) {
    const lastChild = context.firstChild;
    if (lastChild === null) context.firstChild = self;
    else {
      self.nextSibling = lastChild;
      context.firstChild = self;
    }
    if (context.depsTail === null) {
      self.height = context.height;
      recompute(self, true);
    } else {
      self.height = context.height + 1;
      insertIntoHeap(self, dirty);
    }
  } else recompute(self, true);
  return self;
}
function asyncComputed(asyncFn, initialValue) {
  let lastResult = void 0;
  const fn = (prev) => {
    const result = asyncFn(prev);
    lastResult = result;
    const isPromise = result instanceof Promise;
    const iterator = result[Symbol.asyncIterator];
    if (!isPromise && !iterator) return result;
    if (isPromise)
      result
        .then((v) => {
          if (lastResult !== result) return;
          setSignal(self, v);
          stabilize();
        })
        .catch((e) => {
          if (lastResult !== result) return;
          setError(self, e);
          stabilize();
        });
    else
      (async () => {
        try {
          for await (let value of result) {
            if (lastResult !== result) return;
            setSignal(self, value);
            stabilize();
          }
        } catch (error) {
          if (lastResult !== result) return;
          setError(self, error);
          stabilize();
        }
      })();
    throw new NotReadyError(context);
  };
  const self = computed(fn, initialValue);
  return self;
}
function signal(v, firewall = null) {
  if (firewall !== null)
    return (firewall.child = {
      value: v,
      subs: null,
      subsTail: null,
      owner: firewall,
      nextChild: firewall.child,
      asyncFlags: AsyncFlags.None,
      time: clock,
      pendingValue: NOT_PENDING,
    });
  else
    return {
      value: v,
      subs: null,
      subsTail: null,
      asyncFlags: AsyncFlags.None,
      time: clock,
      pendingValue: NOT_PENDING,
    };
}
function recompute(el, create = false) {
  deleteFromHeap(el, el.flags & ReactiveFlags.Zombie ? pending : dirty);
  if (
    el.pendingValue !== NOT_PENDING ||
    el.pendingFirstChild ||
    el.pendingDisposal
  )
    disposeChildren(el);
  else {
    markDisposal(el);
    pendingNodes.push(el);
    el.pendingDisposal = el.disposal;
    el.pendingFirstChild = el.firstChild;
    el.disposal = null;
    el.firstChild = null;
  }
  const oldcontext = context;
  context = el;
  el.depsTail = null;
  el.flags = ReactiveFlags.RecomputingDeps;
  let value = el.pendingValue === NOT_PENDING ? el.value : el.pendingValue;
  let oldHeight = el.height;
  el.time = clock;
  let prevAsyncFlags = el.asyncFlags;
  try {
    value = el.fn(value);
    clearAsyncFlags(el);
  } catch (e) {
    if (e instanceof NotReadyError) {
      asyncNodes.push(e.cause);
      setAsyncFlags(
        el,
        (prevAsyncFlags & ~AsyncFlags.Error) | AsyncFlags.Pending,
      );
    } else setError(el, e);
  }
  el.flags = ReactiveFlags.None;
  context = oldcontext;
  const depsTail = el.depsTail;
  let toRemove = depsTail !== null ? depsTail.nextDep : el.deps;
  if (toRemove !== null) {
    do toRemove = unlinkSubs(toRemove);
    while (toRemove !== null);
    if (depsTail !== null) depsTail.nextDep = null;
    else el.deps = null;
  }
  const valueChanged =
    el.pendingValue === NOT_PENDING
      ? value !== el.value
      : el.pendingValue !== value;
  const asyncFlagsChanged = el.asyncFlags !== prevAsyncFlags;
  if (valueChanged || asyncFlagsChanged) {
    if (valueChanged) {
      if (!create && el.pendingValue === NOT_PENDING) pendingNodes.push(el);
      create ? (el.value = value) : (el.pendingValue = value);
    }
    for (let s = el.subs; s !== null; s = s.nextSub)
      insertIntoHeap(
        s.sub,
        s.sub.flags & ReactiveFlags.Zombie ? pending : dirty,
      );
  } else if (el.height != oldHeight)
    for (let s = el.subs; s !== null; s = s.nextSub)
      insertIntoHeapHeight(
        s.sub,
        s.sub.flags & ReactiveFlags.Zombie ? pending : dirty,
      );
}
function updateIfNecessary(el) {
  if (el.flags & ReactiveFlags.Check)
    for (let d = el.deps; d; d = d.nextDep) {
      const dep1 = d.dep;
      const dep = "owner" in dep1 ? dep1.owner : dep1;
      if ("fn" in dep) updateIfNecessary(dep);
      if (el.flags & ReactiveFlags.Dirty) break;
    }
  if (el.flags & ReactiveFlags.Dirty) recompute(el);
  el.flags = ReactiveFlags.None;
}
function unlinkSubs(link$1) {
  const dep = link$1.dep;
  const nextDep = link$1.nextDep;
  const nextSub = link$1.nextSub;
  const prevSub = link$1.prevSub;
  if (nextSub !== null) nextSub.prevSub = prevSub;
  else dep.subsTail = prevSub;
  if (prevSub !== null) prevSub.nextSub = nextSub;
  else dep.subs = nextSub;
  return nextDep;
}
function link(dep, sub) {
  const prevDep = sub.depsTail;
  if (prevDep !== null && prevDep.dep === dep) return;
  let nextDep = null;
  const isRecomputing = sub.flags & ReactiveFlags.RecomputingDeps;
  if (isRecomputing) {
    nextDep = prevDep !== null ? prevDep.nextDep : sub.deps;
    if (nextDep !== null && nextDep.dep === dep) {
      sub.depsTail = nextDep;
      return;
    }
  }
  const prevSub = dep.subsTail;
  if (
    prevSub !== null &&
    prevSub.sub === sub &&
    (!isRecomputing || isValidLink(prevSub, sub))
  )
    return;
  const newLink =
    (sub.depsTail =
    dep.subsTail =
      {
        dep,
        sub,
        nextDep,
        prevSub,
        nextSub: null,
      });
  if (prevDep !== null) prevDep.nextDep = newLink;
  else sub.deps = newLink;
  if (prevSub !== null) prevSub.nextSub = newLink;
  else dep.subs = newLink;
}
function isValidLink(checkLink, sub) {
  const depsTail = sub.depsTail;
  if (depsTail !== null) {
    let link$1 = sub.deps;
    do {
      if (link$1 === checkLink) return true;
      if (link$1 === depsTail) break;
      link$1 = link$1.nextDep;
    } while (link$1 !== null);
  }
  return false;
}
function read(el, c = context) {
  if (c) {
    link(el, c);
    const owner = "owner" in el ? el.owner : el;
    if ("fn" in owner) {
      const isZombie = el.flags & ReactiveFlags.Zombie;
      if (owner.height >= (isZombie ? pending.min : dirty.min)) {
        markNode(c);
        markHeap(isZombie ? pending : dirty);
        updateIfNecessary(owner);
      }
      const height = owner.height;
      if (height >= c.height) c.height = height + 1;
    }
  }
  if (el.asyncFlags & AsyncFlags.Pending) {
    if ((c && !stale) || el.asyncFlags & AsyncFlags.Uninitialized)
      throw new NotReadyError(el);
  }
  if (el.asyncFlags & AsyncFlags.Error)
    if (el.time < clock) {
      recompute(el, true);
      return read(el);
    } else throw el.error;
  return !c ||
    (stale &&
      transition?.pendingNodes.includes(el) &&
      !transitionComplete(transition)) ||
    el.pendingValue === NOT_PENDING
    ? el.value
    : el.pendingValue;
}
function setSignal(el, v) {
  const valueChanged =
    el.pendingValue === NOT_PENDING ? el.value !== v : el.pendingValue !== v;
  if (!valueChanged && !el.asyncFlags) return;
  if (valueChanged) {
    if (el.pendingValue === NOT_PENDING) pendingNodes.push(el);
    el.pendingValue = v;
  }
  clearAsyncFlags(el);
  el.time = clock;
  for (let link$1 = el.subs; link$1 !== null; link$1 = link$1.nextSub)
    insertIntoHeap(
      link$1.sub,
      link$1.sub.flags & ReactiveFlags.Zombie ? pending : dirty,
    );
}
function setAsyncFlags(signal$1, flags, error = null) {
  signal$1.asyncFlags = flags;
  signal$1.error = error;
}
function setError(signal$1, error) {
  setAsyncFlags(signal$1, AsyncFlags.Error | AsyncFlags.Uninitialized, error);
}
function clearAsyncFlags(signal$1) {
  setAsyncFlags(signal$1, AsyncFlags.None);
}
function markNode(el, newState = ReactiveFlags.Dirty) {
  const flags = el.flags;
  if ((flags & (ReactiveFlags.Check | ReactiveFlags.Dirty)) >= newState) return;
  el.flags = (flags & ~(ReactiveFlags.Check | ReactiveFlags.Dirty)) | newState;
  for (let link$1 = el.subs; link$1 !== null; link$1 = link$1.nextSub)
    markNode(link$1.sub, ReactiveFlags.Check);
  if (el.child !== null)
    for (let child = el.child; child !== null; child = child.nextChild)
      for (let link$1 = child.subs; link$1 !== null; link$1 = link$1.nextSub)
        markNode(link$1.sub, ReactiveFlags.Check);
}
function markHeap(heap) {
  if (markedHeap) return;
  markedHeap = true;
  for (let i = 0; i <= heap.max; i++)
    for (let el = heap.heap[i]; el !== void 0; el = el.nextHeap)
      if (el.flags & ReactiveFlags.InHeap) markNode(el);
}
function adjustHeight(el, heap) {
  deleteFromHeap(el, heap);
  let newHeight = el.height;
  for (let d = el.deps; d; d = d.nextDep) {
    const dep1 = d.dep;
    const dep = "owner" in dep1 ? dep1.owner : dep1;
    if ("fn" in dep) {
      if (dep.height >= newHeight) newHeight = dep.height + 1;
    }
  }
  if (el.height !== newHeight) {
    el.height = newHeight;
    for (let s = el.subs; s !== null; s = s.nextSub)
      insertIntoHeapHeight(s.sub, heap);
  }
}
function stabilize() {
  markedHeap = false;
  runHeap(dirty);
  if (asyncNodes.length > 0) {
    transition = {
      time: clock,
      asyncNodes,
      pendingNodes,
    };
    runHeap(pending);
    asyncNodes = [];
    pendingNodes = [];
    clock++;
    return;
  }
  if (transition && transitionComplete(transition)) {
    pendingNodes.push(...transition.pendingNodes);
    transition = null;
  }
  for (let i = 0; i < pendingNodes.length; i++) {
    const n = pendingNodes[i];
    if (n.pendingValue !== NOT_PENDING) {
      n.value = n.pendingValue;
      n.pendingValue = NOT_PENDING;
    }
    if (n.fn) disposeChildren(n, true);
  }
  pendingNodes.length = 0;
  clock++;
}
function runHeap(heap) {
  for (heap.min = 0; heap.min <= heap.max; heap.min++) {
    let el = heap.heap[heap.min];
    while (el !== void 0) {
      if (el.flags & ReactiveFlags.InHeap) recompute(el);
      else adjustHeight(el, heap);
      el = heap.heap[heap.min];
    }
  }
  heap.max = 0;
}
function transitionComplete(transition$1) {
  let done = true;
  for (let i = 0; i < transition$1.asyncNodes.length; i++)
    if (transition$1.asyncNodes[i].asyncFlags & AsyncFlags.Pending) {
      done = false;
      break;
    }
  return done;
}
function onCleanup(fn) {
  if (!context) return fn;
  const node = context;
  if (!node.disposal) node.disposal = fn;
  else if (Array.isArray(node.disposal)) node.disposal.push(fn);
  else node.disposal = [node.disposal, fn];
  return fn;
}
function markDisposal(el) {
  let child = el.firstChild;
  while (child) {
    child.flags |= ReactiveFlags.Zombie;
    if (child.flags & ReactiveFlags.InHeap) {
      deleteFromHeap(child, dirty);
      insertIntoHeap(child, pending);
    }
    markDisposal(child);
    child = child.nextSibling;
  }
}
function disposeChildren(node, zombie) {
  let child = zombie ? node.pendingFirstChild : node.firstChild;
  while (child) {
    const nextChild = child.nextSibling;
    if (child.deps) {
      const n = child;
      deleteFromHeap(n, n.flags & ReactiveFlags.Zombie ? pending : dirty);
      let toRemove = n.deps;
      do toRemove = unlinkSubs(toRemove);
      while (toRemove !== null);
      n.deps = null;
      n.depsTail = null;
      n.flags = ReactiveFlags.None;
    }
    disposeChildren(child);
    child = nextChild;
  }
  if (zombie) node.pendingFirstChild = null;
  else {
    node.firstChild = null;
    node.nextSibling = null;
  }
  runDisposal(node, zombie);
}
function runDisposal(node, zombie) {
  let disposal = zombie ? node.pendingDisposal : node.disposal;
  if (!disposal || disposal === NOT_PENDING) return;
  if (Array.isArray(disposal))
    for (let i = 0; i < disposal.length; i++) {
      const callable = disposal[i];
      callable.call(callable);
    }
  else disposal.call(disposal);
  zombie ? (node.pendingDisposal = null) : (node.disposal = null);
}
function getContext() {
  return context;
}
function runWithOwner(owner, fn) {
  const oldContext = context;
  context = owner;
  try {
    return fn();
  } finally {
    context = oldContext;
  }
}
function latest(fn) {
  const prevStale = stale;
  stale = true;
  try {
    return fn();
  } finally {
    stale = prevStale;
  }
}

//#endregion
export {
  AsyncFlags,
  NotReadyError,
  ReactiveFlags,
  asyncComputed,
  computed,
  getContext,
  increaseHeapSize,
  latest,
  onCleanup,
  read,
  runWithOwner,
  setSignal,
  signal,
  stabilize,
};
