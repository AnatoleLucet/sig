// src/core/error.ts
var NotReadyError = class extends Error {
  constructor(cause) {
    super();
    this.cause = cause;
  }
};
var NoOwnerError = class extends Error {
  constructor() {
    super("Context can only be accessed under a reactive root.");
  }
};
var ContextNotFoundError = class extends Error {
  constructor() {
    super(
      "Context must either be created with a default value or a value must be provided before accessing it.",
    );
  }
};

// src/core/constants.ts
var NOT_PENDING = {};
var SUPPORTS_PROXY = typeof Proxy === "function";

// src/core/heap.ts
function actualInsertIntoHeap(n, heap) {
  const height = n._height;
  const heapAtHeight = heap._heap[height];
  if (heapAtHeight === void 0) {
    heap._heap[height] = n;
  } else {
    const tail = heapAtHeight._prevHeap;
    tail._nextHeap = n;
    n._prevHeap = tail;
    heapAtHeight._prevHeap = n;
  }
  if (height > heap._max) {
    heap._max = height;
  }
}
function insertIntoHeap(n, heap) {
  let flags = n._flags;
  if (flags & (8 /* InHeap */ | 4) /* RecomputingDeps */) return;
  if (flags & 1 /* Check */) {
    n._flags =
      (flags & ~((1 /* Check */ | 2) /* Dirty */)) |
      2 /* Dirty */ |
      8 /* InHeap */;
  } else n._flags = flags | 8 /* InHeap */;
  if (!((flags & 16) /* InHeapHeight */)) {
    actualInsertIntoHeap(n, heap);
  }
}
function insertIntoHeapHeight(n, heap) {
  let flags = n._flags;
  if (
    flags &
    (8 /* InHeap */ | 4 /* RecomputingDeps */ | 16) /* InHeapHeight */
  )
    return;
  n._flags = flags | 16 /* InHeapHeight */;
  actualInsertIntoHeap(n, heap);
}
function deleteFromHeap(n, heap) {
  const flags = n._flags;
  if (!(flags & (8 /* InHeap */ | 16) /* InHeapHeight */)) return;
  n._flags = flags & ~((8 /* InHeap */ | 16) /* InHeapHeight */);
  const height = n._height;
  if (n._prevHeap === n) {
    heap._heap[height] = void 0;
  } else {
    const next = n._nextHeap;
    const dhh = heap._heap[height];
    const end = next ?? dhh;
    if (n === dhh) {
      heap._heap[height] = next;
    } else {
      n._prevHeap._nextHeap = next;
    }
    end._prevHeap = n._prevHeap;
  }
  n._prevHeap = n;
  n._nextHeap = void 0;
}
function markHeap(heap) {
  if (heap._marked) return;
  heap._marked = true;
  for (let i = 0; i <= heap._max; i++) {
    for (let el = heap._heap[i]; el !== void 0; el = el._nextHeap) {
      if (el._flags & 8 /* InHeap */) markNode(el);
    }
  }
}
function markNode(el, newState = 2 /* Dirty */) {
  const flags = el._flags;
  if ((flags & (1 /* Check */ | 2) /* Dirty */) >= newState) return;
  el._flags = (flags & ~((1 /* Check */ | 2) /* Dirty */)) | newState;
  for (let link2 = el._subs; link2 !== null; link2 = link2._nextSub) {
    markNode(link2._sub, 1 /* Check */);
  }
  if (el._child !== null) {
    for (let child = el._child; child !== null; child = child._nextChild) {
      for (let link2 = child._subs; link2 !== null; link2 = link2._nextSub) {
        markNode(link2._sub, 1 /* Check */);
      }
    }
  }
}
function runHeap(heap, recompute2) {
  heap._marked = false;
  for (heap._min = 0; heap._min <= heap._max; heap._min++) {
    let el = heap._heap[heap._min];
    while (el !== void 0) {
      if (el._flags & 8 /* InHeap */) recompute2(el);
      else {
        adjustHeight(el, heap);
      }
      el = heap._heap[heap._min];
    }
  }
  heap._max = 0;
}
function adjustHeight(el, heap) {
  deleteFromHeap(el, heap);
  let newHeight = el._height;
  for (let d = el._deps; d; d = d._nextDep) {
    const dep1 = d._dep;
    const dep = "_owner" in dep1 ? dep1._owner : dep1;
    if ("_fn" in dep) {
      if (dep._height >= newHeight) {
        newHeight = dep._height + 1;
      }
    }
  }
  if (el._height !== newHeight) {
    el._height = newHeight;
    for (let s = el._subs; s !== null; s = s._nextSub) {
      insertIntoHeapHeight(s._sub, heap);
    }
  }
}

// src/core/scheduler.ts
var clock = 0;
var activeTransition = null;
var unobserved = [];
var transitions = /* @__PURE__ */ new Set();
var scheduled = false;
function schedule() {
  if (scheduled) return;
  scheduled = true;
  if (!globalQueue._running) queueMicrotask(flush);
}
var dirtyQueue = {
  _heap: new Array(2e3).fill(void 0),
  _marked: false,
  _min: 0,
  _max: 0,
};
var pendingQueue = {
  _heap: new Array(2e3).fill(void 0),
  _marked: false,
  _min: 0,
  _max: 0,
};
var Queue = class _Queue {
  _parent = null;
  _running = false;
  _queues = [[], []];
  _children = [];
  _pendingNodes = [];
  created = clock;
  static _update;
  static _dispose;
  enqueue(type, fn) {
    if (type) this._queues[type - 1].push(fn);
    schedule();
  }
  run(type) {
    if (this._queues[type - 1].length) {
      const effects = this._queues[type - 1];
      this._queues[type - 1] = [];
      runQueue(effects, type);
    }
    for (let i = 0; i < this._children.length; i++) {
      this._children[i].run(type);
    }
  }
  flush() {
    if (this._running) return;
    this._running = true;
    try {
      runHeap(dirtyQueue, _Queue._update);
      if (activeTransition) {
        if (!transitionComplete(activeTransition)) {
          runHeap(pendingQueue, _Queue._update);
          globalQueue._pendingNodes = [];
          activeTransition.queues[0].push(...globalQueue._queues[0]);
          activeTransition.queues[1].push(...globalQueue._queues[1]);
          globalQueue._queues = [[], []];
          clock++;
          scheduled = false;
          runPending(activeTransition.pendingNodes, true);
          activeTransition = null;
          return;
        }
        globalQueue._pendingNodes.push(...activeTransition.pendingNodes);
        globalQueue._queues[0].push(...activeTransition.queues[0]);
        globalQueue._queues[1].push(...activeTransition.queues[1]);
        transitions.delete(activeTransition);
        activeTransition = null;
        if (runPending(globalQueue._pendingNodes, false))
          runHeap(dirtyQueue, _Queue._update);
      } else if (transitions.size) runHeap(pendingQueue, _Queue._update);
      for (let i = 0; i < globalQueue._pendingNodes.length; i++) {
        const n = globalQueue._pendingNodes[i];
        if (n._pendingValue !== NOT_PENDING) {
          n._value = n._pendingValue;
          n._pendingValue = NOT_PENDING;
        }
        if (n._fn) _Queue._dispose(n, true);
      }
      globalQueue._pendingNodes.length = 0;
      clock++;
      scheduled = false;
      this.run(1 /* Render */);
      this.run(2 /* User */);
    } finally {
      this._running = false;
      unobserved.length && notifyUnobserved();
    }
  }
  addChild(child) {
    this._children.push(child);
    child._parent = this;
  }
  removeChild(child) {
    const index = this._children.indexOf(child);
    if (index >= 0) {
      this._children.splice(index, 1);
      child._parent = null;
    }
  }
  notify(node, mask, flags) {
    if (mask & 1 /* Pending */) {
      if (flags & 1 /* Pending */) {
        if (
          activeTransition &&
          !activeTransition.asyncNodes.includes(node._error.cause)
        )
          activeTransition.asyncNodes.push(node._error.cause);
      }
      return true;
    }
    if (this._parent) return this._parent.notify(node, mask, flags);
    return false;
  }
  initTransition(node) {
    if (activeTransition && activeTransition.time === clock) return;
    if (!activeTransition) {
      activeTransition = node._transition ?? {
        time: clock,
        pendingNodes: [],
        asyncNodes: [],
        queues: [[], []],
      };
    }
    transitions.add(activeTransition);
    activeTransition.time = clock;
    for (let i = 0; i < globalQueue._pendingNodes.length; i++) {
      const n = globalQueue._pendingNodes[i];
      n._transition = activeTransition;
      activeTransition.pendingNodes.push(n);
    }
    globalQueue._pendingNodes = activeTransition.pendingNodes;
  }
};
function runPending(pendingNodes, value) {
  let needsReset = false;
  const p = pendingNodes.slice();
  for (let i = 0; i < p.length; i++) {
    const n = p[i];
    n._transition = activeTransition;
    if (n._pendingCheck) {
      n._pendingCheck._set(value);
      needsReset = true;
    }
  }
  return needsReset;
}
var globalQueue = new Queue();
function flush() {
  let count = 0;
  while (scheduled) {
    if (++count === 1e5) throw new Error("Potential Infinite Loop Detected.");
    globalQueue.flush();
  }
}
function runQueue(queue, type) {
  for (let i = 0; i < queue.length; i++) queue[i](type);
}
function transitionComplete(transition) {
  let done = true;
  for (let i = 0; i < transition.asyncNodes.length; i++) {
    if (transition.asyncNodes[i]._statusFlags & 1 /* Pending */) {
      done = false;
      break;
    }
  }
  return done;
}
function notifyUnobserved() {
  for (let i = 0; i < unobserved.length; i++) {
    const source = unobserved[i];
    if (!source._subs) unobserved[i]._unobserved?.();
  }
  unobserved = [];
}

// src/core/core.ts
Queue._update = recompute;
Queue._dispose = disposeChildren;
var tracking = true;
var stale = false;
var pendingValueCheck = false;
var pendingCheck = null;
var context = null;
var defaultContext = {};
function recompute(el, create = false) {
  deleteFromHeap(el, el._flags & 32 /* Zombie */ ? pendingQueue : dirtyQueue);
  if (
    el._pendingValue !== NOT_PENDING ||
    el._pendingFirstChild ||
    el._pendingDisposal
  )
    disposeChildren(el);
  else {
    markDisposal(el);
    globalQueue._pendingNodes.push(el);
    el._pendingDisposal = el._disposal;
    el._pendingFirstChild = el._firstChild;
    el._disposal = null;
    el._firstChild = null;
  }
  const oldcontext = context;
  context = el;
  el._depsTail = null;
  el._flags = 4 /* RecomputingDeps */;
  let value = el._pendingValue === NOT_PENDING ? el._value : el._pendingValue;
  let oldHeight = el._height;
  el._time = clock;
  let prevStatusFlags = el._statusFlags;
  let prevError = el._error;
  clearStatusFlags(el);
  try {
    value = el._fn(value);
  } catch (e) {
    if (e instanceof NotReadyError) {
      setStatusFlags(
        el,
        (prevStatusFlags & ~2) /* Error */ | 1 /* Pending */,
        e,
      );
    } else {
      setError(el, e);
    }
  }
  el._notifyQueue?.();
  el._flags = 0 /* None */;
  context = oldcontext;
  const depsTail = el._depsTail;
  let toRemove = depsTail !== null ? depsTail._nextDep : el._deps;
  if (toRemove !== null) {
    do {
      toRemove = unlinkSubs(toRemove);
    } while (toRemove !== null);
    if (depsTail !== null) {
      depsTail._nextDep = null;
    } else {
      el._deps = null;
    }
  }
  const valueChanged =
    !el._equals ||
    !el._equals(
      el._pendingValue === NOT_PENDING ? el._value : el._pendingValue,
      value,
    );
  const statusFlagsChanged =
    el._statusFlags !== prevStatusFlags || el._error !== prevError;
  if (valueChanged || statusFlagsChanged) {
    if (valueChanged) {
      if (create || el._optimistic || el._type) el._value = value;
      else {
        if (el._pendingValue === NOT_PENDING)
          globalQueue._pendingNodes.push(el);
        el._pendingValue = value;
      }
      if (el._pendingSignal) el._pendingSignal._set(value);
    }
    for (let s = el._subs; s !== null; s = s._nextSub) {
      insertIntoHeap(
        s._sub,
        s._sub._flags & 32 /* Zombie */ ? pendingQueue : dirtyQueue,
      );
    }
  } else if (el._height != oldHeight) {
    for (let s = el._subs; s !== null; s = s._nextSub) {
      insertIntoHeapHeight(
        s._sub,
        s._sub._flags & 32 /* Zombie */ ? pendingQueue : dirtyQueue,
      );
    }
  }
}
function updateIfNecessary(el) {
  if (el._flags & 1 /* Check */) {
    for (let d = el._deps; d; d = d._nextDep) {
      const dep1 = d._dep;
      const dep = "_owner" in dep1 ? dep1._owner : dep1;
      if ("_fn" in dep) {
        updateIfNecessary(dep);
      }
      if (el._flags & 2 /* Dirty */) {
        break;
      }
    }
  }
  if (el._flags & 2 /* Dirty */) {
    recompute(el);
  }
  el._flags = 0 /* None */;
}
function unlinkSubs(link2) {
  const dep = link2._dep;
  const nextDep = link2._nextDep;
  const nextSub = link2._nextSub;
  const prevSub = link2._prevSub;
  if (nextSub !== null) {
    nextSub._prevSub = prevSub;
  } else {
    dep._subsTail = prevSub;
  }
  if (prevSub !== null) {
    prevSub._nextSub = nextSub;
  } else {
    dep._subs = nextSub;
  }
  return nextDep;
}
function link(dep, sub) {
  const prevDep = sub._depsTail;
  if (prevDep !== null && prevDep._dep === dep) {
    return;
  }
  let nextDep = null;
  const isRecomputing = sub._flags & 4; /* RecomputingDeps */
  if (isRecomputing) {
    nextDep = prevDep !== null ? prevDep._nextDep : sub._deps;
    if (nextDep !== null && nextDep._dep === dep) {
      sub._depsTail = nextDep;
      return;
    }
  }
  const prevSub = dep._subsTail;
  if (
    prevSub !== null &&
    prevSub._sub === sub &&
    (!isRecomputing || isValidLink(prevSub, sub))
  ) {
    return;
  }
  const newLink =
    (sub._depsTail =
    dep._subsTail =
      {
        _dep: dep,
        _sub: sub,
        _nextDep: nextDep,
        _prevSub: prevSub,
        _nextSub: null,
      });
  if (prevDep !== null) {
    prevDep._nextDep = newLink;
  } else {
    sub._deps = newLink;
  }
  if (prevSub !== null) {
    prevSub._nextSub = newLink;
  } else {
    dep._subs = newLink;
  }
}
function isValidLink(checkLink, sub) {
  const depsTail = sub._depsTail;
  if (depsTail !== null) {
    let link2 = sub._deps;
    do {
      if (link2 === checkLink) {
        return true;
      }
      if (link2 === depsTail) {
        break;
      }
      link2 = link2._nextDep;
    } while (link2 !== null);
  }
  return false;
}
function setStatusFlags(signal2, flags, error = null) {
  signal2._statusFlags = flags;
  signal2._error = error;
}
function setError(signal2, error) {
  setStatusFlags(signal2, 2 /* Error */ | 4 /* Uninitialized */, error);
}
function clearStatusFlags(signal2) {
  setStatusFlags(signal2, 0 /* None */);
}
function markDisposal(el) {
  let child = el._firstChild;
  while (child) {
    child._flags |= 32 /* Zombie */;
    const inHeap = child._flags & 8; /* InHeap */
    if (inHeap) {
      deleteFromHeap(child, dirtyQueue);
      insertIntoHeap(child, pendingQueue);
    }
    markDisposal(child);
    child = child._nextSibling;
  }
}
function disposeChildren(node, zombie) {
  let child = zombie ? node._pendingFirstChild : node._firstChild;
  while (child) {
    const nextChild = child._nextSibling;
    if (child._deps) {
      const n = child;
      deleteFromHeap(n, n._flags & 32 /* Zombie */ ? pendingQueue : dirtyQueue);
      let toRemove = n._deps;
      do {
        toRemove = unlinkSubs(toRemove);
      } while (toRemove !== null);
      n._deps = null;
      n._depsTail = null;
      n._flags = 0 /* None */;
    }
    disposeChildren(child);
    child = nextChild;
  }
  if (zombie) {
    node._pendingFirstChild = null;
  } else {
    node._firstChild = null;
    node._nextSibling = null;
  }
  runDisposal(node, zombie);
}
function runDisposal(node, zombie) {
  let disposal = zombie ? node._pendingDisposal : node._disposal;
  if (!disposal || disposal === NOT_PENDING) return;
  if (Array.isArray(disposal)) {
    for (let i = 0; i < disposal.length; i++) {
      const callable = disposal[i];
      callable.call(callable);
    }
  } else {
    disposal.call(disposal);
  }
  zombie ? (node._pendingDisposal = null) : (node._disposal = null);
}
function withOptions(obj, options) {
  obj._name = options?.name ?? (obj._fn ? "computed" : "signal");
  obj._id = options?.id;
  obj._equals = options?.equals !== void 0 ? options.equals : isEqual;
  obj._pureWrite = !!options?.pureWrite;
  obj._unobserved = options?.unobserved;
  if (options?._internal) Object.assign(obj, options._internal);
  return obj;
}
function getNextChildId(owner) {
  if (owner._id != null) return formatId(owner._id, owner._childCount++);
  throw new Error("Cannot get child id from owner without an id");
}
function formatId(prefix, id) {
  const num = id.toString(36),
    len = num.length - 1;
  return prefix + (len ? String.fromCharCode(64 + len) : "") + num;
}
function computed(fn, initialValue, options) {
  const self = withOptions(
    {
      _disposal: null,
      _queue: globalQueue,
      _context: defaultContext,
      _childCount: 0,
      _fn: fn,
      _value: initialValue,
      _height: 0,
      _child: null,
      _nextHeap: void 0,
      _prevHeap: null,
      _deps: null,
      _depsTail: null,
      _subs: null,
      _subsTail: null,
      _parent: context,
      _nextSibling: null,
      _firstChild: null,
      _flags: 0 /* None */,
      _statusFlags: 4 /* Uninitialized */,
      _time: clock,
      _pendingValue: NOT_PENDING,
      _pendingDisposal: null,
      _pendingFirstChild: null,
    },
    options,
  );
  self._prevHeap = self;
  const parent = context?._root ? context._parentComputed : context;
  if (context) {
    context._queue && (self._queue = context._queue);
    context._context && (self._context = context._context);
    const lastChild = context._firstChild;
    if (lastChild === null) {
      context._firstChild = self;
    } else {
      self._nextSibling = lastChild;
      context._firstChild = self;
    }
    if (parent) {
      if (parent._depsTail === null || options._forceRun) {
        self._height = parent._height;
        recompute(self, true);
      } else {
        self._height = parent._height + 1;
        insertIntoHeap(self, dirtyQueue);
      }
    }
  } else {
    recompute(self, true);
  }
  return self;
}
function asyncComputed(asyncFn, initialValue, options) {
  let lastResult = void 0;
  let refreshing = false;
  const fn = (prev) => {
    const result = asyncFn(prev, refreshing);
    refreshing = false;
    lastResult = result;
    const isPromise = result instanceof Promise;
    const iterator = result[Symbol.asyncIterator];
    if (!isPromise && !iterator) {
      return result;
    }
    if (isPromise) {
      result
        .then((v) => {
          if (lastResult !== result) return;
          globalQueue.initTransition(self);
          setSignal(self, v);
          flush();
        })
        .catch((e) => {
          if (lastResult !== result) return;
          globalQueue.initTransition(self);
          setError(self, e);
          flush();
        });
    } else {
      (async () => {
        try {
          for await (let value of result) {
            if (lastResult !== result) return;
            globalQueue.initTransition(self);
            setSignal(self, value);
            flush();
          }
        } catch (error) {
          if (lastResult !== result) return;
          globalQueue.initTransition(self);
          setError(self, error);
          flush();
        }
      })();
    }
    globalQueue.initTransition(context);
    throw new NotReadyError(context);
  };
  const self = computed(fn, initialValue, options);
  self._refresh = () => {
    refreshing = true;
    recompute(self);
    flush();
  };
  return self;
}
function signal(v, options, firewall = null) {
  if (firewall !== null) {
    return (firewall._child = withOptions(
      {
        _value: v,
        _subs: null,
        _subsTail: null,
        _owner: firewall,
        _nextChild: firewall._child,
        _statusFlags: 0 /* None */,
        _time: clock,
        _pendingValue: NOT_PENDING,
      },
      options,
    ));
  } else {
    return withOptions(
      {
        _value: v,
        _subs: null,
        _subsTail: null,
        _statusFlags: 0 /* None */,
        _time: clock,
        _pendingValue: NOT_PENDING,
      },
      options,
    );
  }
}
function isEqual(a, b) {
  return a === b;
}
function untrack(fn) {
  if (!tracking) return fn();
  tracking = false;
  try {
    return fn();
  } finally {
    tracking = true;
  }
}
function read(el) {
  let c = context;
  if (c?._root) c = c._parentComputed;
  if (c && tracking) {
    link(el, c);
    const owner = "_owner" in el ? el._owner : el;
    if ("_fn" in owner) {
      const isZombie = el._flags & 32; /* Zombie */
      if (owner._height >= (isZombie ? pendingQueue._min : dirtyQueue._min)) {
        markNode(c);
        markHeap(isZombie ? pendingQueue : dirtyQueue);
        updateIfNecessary(owner);
      }
      const height = owner._height;
      if (height >= c._height) {
        c._height = height + 1;
      }
    }
  }
  if (pendingCheck) {
    if (!el._pendingCheck) {
      el._pendingCheck = signal(
        (el._statusFlags & 1) /* Pending */ !== 0 || !!el._transition || false,
      );
      el._pendingCheck._optimistic = true;
      el._pendingCheck._set = (v) => setSignal(el._pendingCheck, v);
    }
    const prev = pendingCheck;
    pendingCheck = null;
    prev._value = read(el._pendingCheck) || prev._value;
    pendingCheck = prev;
  }
  if (pendingValueCheck) {
    if (!el._pendingSignal) {
      el._pendingSignal = signal(
        el._pendingValue === NOT_PENDING ? el._value : el._pendingValue,
      );
      el._pendingSignal._optimistic = true;
      el._pendingSignal._set = (v) =>
        queueMicrotask(() =>
          queueMicrotask(() => setSignal(el._pendingSignal, v)),
        );
    }
    pendingValueCheck = false;
    try {
      return read(el._pendingSignal);
    } finally {
      pendingValueCheck = true;
    }
  }
  if (el._statusFlags & 1 /* Pending */) {
    if ((c && !stale) || el._statusFlags & 4 /* Uninitialized */)
      throw el._error;
    else if (c && stale && !pendingCheck) {
      setStatusFlags(c, c._statusFlags | 1, el._error);
    }
  }
  if (el._statusFlags & 2 /* Error */) {
    if (el._time < clock) {
      recompute(el, true);
      return read(el);
    } else {
      throw el._error;
    }
  }
  return !c ||
    el._pendingValue === NOT_PENDING ||
    (stale &&
      !pendingCheck &&
      el._transition &&
      activeTransition !== el._transition)
    ? el._value
    : el._pendingValue;
}
function setSignal(el, v) {
  if (typeof v === "function") {
    v = v(el._pendingValue === NOT_PENDING ? el._value : el._pendingValue);
  }
  const valueChanged =
    !el._equals ||
    !el._equals(
      el._pendingValue === NOT_PENDING ? el._value : el._pendingValue,
      v,
    );
  if (!valueChanged && !el._statusFlags) return;
  if (valueChanged) {
    if (el._optimistic) el._value = v;
    else {
      if (el._pendingValue === NOT_PENDING) globalQueue._pendingNodes.push(el);
      el._pendingValue = v;
    }
    if (el._pendingSignal) el._pendingSignal._set(v);
  }
  clearStatusFlags(el);
  el._time = clock;
  for (let link2 = el._subs; link2 !== null; link2 = link2._nextSub) {
    insertIntoHeap(
      link2._sub,
      link2._sub._flags & 32 /* Zombie */ ? pendingQueue : dirtyQueue,
    );
  }
  if (el._subs) schedule();
}
function getObserver() {
  return tracking ? context : null;
}
function getOwner() {
  return context;
}
function onCleanup(fn) {
  if (!context) return fn;
  const node = context;
  if (!node._disposal) {
    node._disposal = fn;
  } else if (Array.isArray(node._disposal)) {
    node._disposal.push(fn);
  } else {
    node._disposal = [node._disposal, fn];
  }
  return fn;
}
function createRoot(init, options) {
  const parent = context;
  const owner = {
    _root: true,
    _parentComputed: parent?._root ? parent._parentComputed : parent,
    _disposal: null,
    _id: options?.id ?? (parent?._id ? getNextChildId(parent) : void 0),
    _queue: parent?._queue ?? globalQueue,
    _context: parent?._context || defaultContext,
    _childCount: 0,
    _pendingDisposal: null,
    _pendingFirstChild: null,
    _parent: parent,
  };
  if (parent) {
    const lastChild = parent._firstChild;
    if (lastChild === null) {
      parent._firstChild = owner;
    } else {
      owner._nextSibling = lastChild;
      parent._firstChild = owner;
    }
  }
  return runWithOwner(
    owner,
    !init.length ? init : () => init(() => disposeChildren(owner)),
  );
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
function staleValues(fn, set = true) {
  const prevStale = stale;
  stale = set;
  try {
    return fn();
  } finally {
    stale = prevStale;
  }
}
function pending(fn) {
  const prevLatest = pendingValueCheck;
  pendingValueCheck = true;
  try {
    return staleValues(fn, false);
  } finally {
    pendingValueCheck = prevLatest;
  }
}
function isPending(fn, loadingValue) {
  const current = pendingCheck;
  pendingCheck = { _value: false };
  try {
    staleValues(fn);
    return pendingCheck._value;
  } catch (err) {
    if (!(err instanceof NotReadyError)) return false;
    if (loadingValue !== void 0) return loadingValue;
    throw err;
  } finally {
    pendingCheck = current;
  }
}

// src/core/context.ts
function createContext(defaultValue, description) {
  return { id: Symbol(description), defaultValue };
}
function getContext(context2, owner = getOwner()) {
  if (!owner) {
    throw new NoOwnerError();
  }
  const value = hasContext(context2, owner)
    ? owner._context[context2.id]
    : context2.defaultValue;
  if (isUndefined(value)) {
    throw new ContextNotFoundError();
  }
  return value;
}
function setContext(context2, value, owner = getOwner()) {
  if (!owner) {
    throw new NoOwnerError();
  }
  owner._context = {
    ...owner._context,
    [context2.id]: isUndefined(value) ? context2.defaultValue : value,
  };
}
function hasContext(context2, owner) {
  return !isUndefined(owner?._context[context2.id]);
}
function isUndefined(value) {
  return typeof value === "undefined";
}

// src/core/effect.ts
function effect(compute, effect2, error, initialValue, options) {
  let initialized = false;
  const node = computed(compute, initialValue, {
    ...options,
    _forceRun: true,
    equals(prev, val) {
      const equal = isEqual(prev, val);
      if (initialized) {
        node._modified = !equal;
        if (!equal && !((node._statusFlags & 2) /* Error */)) {
          node._queue.enqueue(node._type, runEffect.bind(node));
        }
      }
      return equal;
    },
    _internal: {
      _modified: true,
      _prevValue: initialValue,
      _effectFn: effect2,
      _errorFn: error,
      _cleanup: void 0,
      _queue: getOwner()?._queue ?? globalQueue,
      _type: options?.render ? 1 /* Render */ : 2 /* User */,
      _notifyQueue() {
        this._type === 1 /* Render */ &&
          this._queue.notify(
            this,
            1 /* Pending */ | 2 /* Error */,
            this._statusFlags,
          );
      },
    },
  });
  initialized = true;
  if (node._type === 1 /* Render */) {
    node._fn = (p) =>
      !((node._statusFlags & 2) /* Error */)
        ? staleValues(() => compute(p))
        : compute(p);
  }
  !options?.defer &&
    !(node._statusFlags & (2 /* Error */ | 1) /* Pending */) &&
    (node._type === 2 /* User */
      ? node._queue.enqueue(node._type, runEffect.bind(node))
      : runEffect.call(node));
  onCleanup(() => node._cleanup?.());
  if (!node._parent)
    console.warn(
      "Effects created outside a reactive context will never be disposed",
    );
}
function runEffect() {
  if (!this._modified) return;
  this._cleanup?.();
  this._cleanup = void 0;
  try {
    this._cleanup = this._effectFn(this._value, this._prevValue);
  } catch (error) {
    this._queue.notify(this, 1 /* Pending */, 0);
    if (this._type === 2 /* User */) {
      try {
        return this._errorFn
          ? this._errorFn(error, () => {
              this._cleanup?.();
              this._cleanup = void 0;
            })
          : console.error(error);
      } catch (e) {
        error = e;
      }
    }
    if (!this._queue.notify(this, 2 /* Error */, 2 /* Error */)) throw error;
  } finally {
    this._prevValue = this._value;
    this._modified = false;
  }
}

// src/signals.ts
function createSignal(first, second, third) {
  if (typeof first === "function") {
    const node2 = computed(first, second, third);
    return [read.bind(null, node2), setSignal.bind(null, node2)];
  }
  const o = getOwner();
  const needsId = o?._id != null;
  const node = signal(
    first,
    needsId ? { id: getNextChildId(o), ...second } : second,
  );
  return [read.bind(null, node), setSignal.bind(null, node)];
}
function createMemo(compute, value, options) {
  let node = computed(compute, value, options);
  return read.bind(null, node);
}
function createAsync(compute, value, options) {
  const node = asyncComputed(compute, value, options);
  const ret = read.bind(null, node);
  ret.refresh = node._refresh;
  return ret;
}
function createEffect(compute, effectFn, value, options) {
  void effect(compute, effectFn.effect || effectFn, effectFn.error, value, {
    ...options,
    name: options?.name ?? "effect",
  });
}
function createRenderEffect(compute, effectFn, value, options) {
  void effect(compute, effectFn, void 0, value, {
    render: true,
    ...{ ...options, name: options?.name ?? "effect" },
  });
}
function createTrackedEffect(compute, options) {}
function createReaction(effect2, options) {}
function resolve(fn) {
  return new Promise((res, rej) => {
    createRoot((dispose) => {
      computed(() => {
        try {
          res(fn());
        } catch (err) {
          if (err instanceof NotReadyError) throw err;
          rej(err);
        }
        dispose();
      });
    });
  });
}
function createOptimistic(first, second, third) {
  return {};
}

export {
  ContextNotFoundError,
  NoOwnerError,
  NotReadyError,
  Queue,
  SUPPORTS_PROXY,
  createAsync,
  createContext,
  createEffect,
  createMemo,
  createOptimistic,
  createReaction,
  createRenderEffect,
  createSignal,
  createTrackedEffect,
  flush,
  getContext,
  getObserver,
  getOwner,
  isEqual,
  isPending,
  onCleanup,
  pending,
  resolve,
  setContext,
  untrack,
};
