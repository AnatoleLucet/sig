/**
 * Ryan's v2 Reactive System - Readable Version
 *
 * A fine-grained reactive system with support for:
 * - Signals (reactive values)
 * - Computed values (derived state)
 * - Effects (side effects)
 * - Async operations with transitions
 * - Context API
 * - Optimistic updates
 */

// ============================================================================
// TYPES & INTERFACES
// ============================================================================

/**
 * Flags indicating the reactive state of a node
 */
enum ReactiveFlags {
  None = 0,
  Check = 1 << 0, // Node needs to check if dependencies changed
  Dirty = 1 << 1, // Node needs recomputation
  RecomputingDeps = 1 << 2, // Currently recomputing dependencies
  InHeap = 1 << 3, // Node is in execution heap
  InHeapHeight = 1 << 4, // Node is in heap for height adjustment
  Zombie = 1 << 5, // Node is marked for disposal
}

/**
 * Flags for async/error state
 */
enum StatusFlags {
  None = 0,
  Pending = 1 << 0, // Async operation in progress
  Error = 1 << 1, // Error occurred
  Uninitialized = 1 << 2, // Never computed yet
}

/**
 * Effect execution types
 */
enum EffectType {
  Render = 1, // Render effects run during render phase
  User = 2, // User effects run after render
}

/**
 * Bidirectional link between a dependency (observable) and subscriber (computed/effect)
 */
interface DependencyLink {
  _dep: ReactiveNode; // The dependency being observed
  _sub: ReactiveNode; // The subscriber observing
  _nextDep: DependencyLink | null; // Next in subscriber's dependency list
  _prevSub: DependencyLink | null; // Previous in dependency's subscriber list
  _nextSub: DependencyLink | null; // Next in dependency's subscriber list
}

/**
 * A reactive node (signal, computed, or effect)
 */
interface ReactiveNode {
  // Value state
  _value: any;
  _pendingValue: any; // Value awaiting commit
  _equals?: (a: any, b: any) => boolean;

  // Dependency graph
  _deps: DependencyLink | null; // List of dependencies
  _depsTail: DependencyLink | null; // Tail of dependency list
  _subs: DependencyLink | null; // List of subscribers
  _subsTail: DependencyLink | null; // Tail of subscriber list

  // Heap membership (for height-ordered execution)
  _nextHeap?: ReactiveNode;
  _prevHeap: ReactiveNode | null;
  _height: number; // Topological depth in dependency graph

  // Tree structure (parent/child relationships)
  _parent: ReactiveNode | null;
  _nextSibling: ReactiveNode | null;
  _firstChild: ReactiveNode | null;

  // Disposal
  _disposal: Function | Function[] | null;
  _pendingDisposal: any;
  _pendingFirstChild: ReactiveNode | null;

  // Execution state
  _fn?: (prevValue: any) => any; // Computation function
  _flags: ReactiveFlags;
  _time: number; // Last update time (clock tick)

  // Async state
  _statusFlags: StatusFlags;
  _error?: any;
  _transition?: Transition;

  // Other
  _name?: string;
  _id?: string;
  _childCount?: number;
  _queue?: Queue;
  _context?: Record<symbol, any>;
  _optimistic?: boolean; // Updates immediately without batching
  _pureWrite?: boolean;
  _unobserved?: () => void;

  // For roots
  _root?: boolean;
  _parentComputed?: ReactiveNode;

  // For effects
  _type?: EffectType;
  _modified?: boolean;
  _prevValue?: any;
  _effectFn?: (value: any, prevValue: any) => void | Function;
  _errorFn?: (error: any, reset: () => void) => void;
  _cleanup?: Function;
  _notifyQueue?: () => void;

  // For optimistic updates
  _pendingCheck?: ReactiveNode;
  _pendingSignal?: ReactiveNode;
  _set?: (value: any) => void;

  // For async
  _refresh?: () => void;

  // For signals with firewall
  _owner?: ReactiveNode;
  _nextChild?: ReactiveNode;
  _child?: ReactiveNode;
}

/**
 * Transition for coordinating async operations
 */
interface Transition {
  time: number; // Clock tick when started
  pendingNodes: ReactiveNode[]; // Nodes with pending values
  asyncNodes: ReactiveNode[]; // Nodes that threw NotReadyError
  queues: [Function[], Function[]]; //queues for render and user effects
}

/**
 * Priority queue ordered by node height
 */
interface Heap {
  _heap: (ReactiveNode | undefined)[];
  _marked: boolean;
  _min: number;
  _max: number;
}

// ============================================================================
// ERRORS
// ============================================================================

/**
 * Thrown when async signal is read before it's ready (suspense-like behavior)
 */
class NotReadyError extends Error {
  cause: any;

  constructor(cause: any) {
    super();
    this.cause = cause;
  }
}

/**
 * Thrown when context is accessed outside a reactive root
 */
class NoOwnerError extends Error {
  constructor() {
    super("Context can only be accessed under a reactive root.");
  }
}

/**
 * Thrown when context value not found
 */
class ContextNotFoundError extends Error {
  constructor() {
    super(
      "Context must either be created with a default value or a value must be provided before accessing it.",
    );
  }
}

// ============================================================================
// CONSTANTS
// ============================================================================

/** Sentinel value for "no pending update" */
const NOT_PENDING = {};

/** Whether Proxy is supported */
const SUPPORTS_PROXY = typeof Proxy === "function";

// ============================================================================
// GLOBAL STATE
// ============================================================================

/** Logical clock for tracking update order */
let clock = 0;

/** Currently active async transition */
let activeTransition: Transition | null = null;

/** Set of all active transitions */
const transitions = new Set<Transition>();

/** Signals that lost all observers */
let unobserved: ReactiveNode[] = [];

/** Whether a flush is scheduled */
let scheduled = false;

/** Whether dependency tracking is enabled */
let tracking = true;

/** Whether to read stale (current) values instead of pending */
let stale = false;

/** Whether to check pending state */
let pendingValueCheck = false;
let pendingCheck: { _value: boolean } | null = null;

/** Currently executing reactive context */
let context: ReactiveNode | null = null;

/** Default context object */
const defaultContext = {};

/**
 * Heap for dirty nodes (need recomputation)
 */
const dirtyQueue: Heap = {
  _heap: new Array(2000).fill(undefined),
  _marked: false,
  _min: 0,
  _max: 0,
};

/**
 * Heap for pending nodes (async operations, zombies)
 */
const pendingQueue: Heap = {
  _heap: new Array(2000).fill(undefined),
  _marked: false,
  _min: 0,
  _max: 0,
};

// ============================================================================
// HEAP OPERATIONS (Priority Queue for Height-Ordered Execution)
// ============================================================================

/**
 * Insert a node into the heap at its height level.
 * Heap is organized as circular linked lists per height.
 */
function actualInsertIntoHeap(node: ReactiveNode, heap: Heap): void {
  const height = node._height;
  const nodesAtHeight = heap._heap[height];

  if (nodesAtHeight === undefined) {
    // First node at this height
    heap._heap[height] = node;
    node._prevHeap = node; // Circular: points to self
    node._nextHeap = undefined;
  } else {
    // Insert at end of circular list
    const tail = nodesAtHeight._prevHeap!;
    tail._nextHeap = node;
    node._prevHeap = tail;
    node._nextHeap = undefined;
    nodesAtHeight._prevHeap = node; // Update head's back pointer
  }

  if (height > heap._max) {
    heap._max = height;
  }
}

/**
 * Mark node as dirty and insert into heap for recomputation
 */
function insertIntoHeap(node: ReactiveNode, heap: Heap): void {
  const flags = node._flags;

  // Already in heap or currently recomputing
  if (flags & (ReactiveFlags.InHeap | ReactiveFlags.RecomputingDeps)) {
    return;
  }

  // Upgrade Check to Dirty
  if (flags & ReactiveFlags.Check) {
    node._flags =
      (flags & ~(ReactiveFlags.Check | ReactiveFlags.Dirty)) |
      ReactiveFlags.Dirty |
      ReactiveFlags.InHeap;
  } else {
    node._flags = flags | ReactiveFlags.InHeap;
  }

  // Don't insert twice if already in heap for height adjustment
  if (!(flags & ReactiveFlags.InHeapHeight)) {
    actualInsertIntoHeap(node, heap);
  }
}

/**
 * Insert node into heap for height adjustment only (not dirty)
 */
function insertIntoHeapHeight(node: ReactiveNode, heap: Heap): void {
  const flags = node._flags;

  if (
    flags &
    (ReactiveFlags.InHeap |
      ReactiveFlags.RecomputingDeps |
      ReactiveFlags.InHeapHeight)
  ) {
    return;
  }

  node._flags = flags | ReactiveFlags.InHeapHeight;
  actualInsertIntoHeap(node, heap);
}

/**
 * Remove node from heap
 */
function deleteFromHeap(node: ReactiveNode, heap: Heap): void {
  const flags = node._flags;

  if (!(flags & (ReactiveFlags.InHeap | ReactiveFlags.InHeapHeight))) {
    return;
  }

  node._flags = flags & ~(ReactiveFlags.InHeap | ReactiveFlags.InHeapHeight);

  const height = node._height;

  // Single node in circular list
  if (node._prevHeap === node) {
    heap._heap[height] = undefined;
  } else {
    // Multiple nodes - update links
    const next = node._nextHeap;
    const head = heap._heap[height];
    const end = next ?? head!;

    if (node === head) {
      // Removing head
      heap._heap[height] = next;
    } else {
      // Removing non-head
      node._prevHeap!._nextHeap = next;
    }

    end._prevHeap = node._prevHeap;
  }

  // Reset node's heap pointers
  node._prevHeap = node;
  node._nextHeap = undefined;
}

/**
 * Mark all nodes in heap as dirty (propagate updates)
 */
function markHeap(heap: Heap): void {
  if (heap._marked) return;
  heap._marked = true;

  for (let i = 0; i <= heap._max; i++) {
    let element = heap._heap[i];
    while (element !== undefined) {
      if (element._flags & ReactiveFlags.InHeap) {
        markNode(element);
      }
      element = element._nextHeap;
    }
  }
}

/**
 * Mark a node and its subscribers as needing updates
 */
function markNode(
  element: ReactiveNode,
  newState: ReactiveFlags = ReactiveFlags.Dirty,
): void {
  const flags = element._flags;

  // Already marked with this state or dirtier
  if ((flags & (ReactiveFlags.Check | ReactiveFlags.Dirty)) >= newState) {
    return;
  }

  element._flags =
    (flags & ~(ReactiveFlags.Check | ReactiveFlags.Dirty)) | newState;

  // Mark all subscribers as needing check
  for (let link = element._subs; link !== null; link = link._nextSub) {
    markNode(link._sub, ReactiveFlags.Check);
  }

  // Also mark subscribers of firewall children
  if (element._child !== null) {
    for (
      let child = element._child;
      child !== null;
      child = child._nextChild!
    ) {
      for (let link = child._subs; link !== null; link = link._nextSub) {
        markNode(link._sub, ReactiveFlags.Check);
      }
    }
  }
}

/**
 * Process all nodes in heap in height order (topological sort)
 */
function runHeap(
  heap: Heap,
  recomputeFunc: (node: ReactiveNode) => void,
): void {
  heap._marked = false;

  for (heap._min = 0; heap._min <= heap._max; heap._min++) {
    let element = heap._heap[heap._min];

    while (element !== undefined) {
      if (element._flags & ReactiveFlags.InHeap) {
        recomputeFunc(element);
      } else {
        // Not dirty, just needs height adjustment
        adjustHeight(element, heap);
      }

      // Re-fetch (may have changed during recompute)
      element = heap._heap[heap._min];
    }
  }

  heap._max = 0;
}

/**
 * Recalculate node's height based on dependency heights
 */
function adjustHeight(element: ReactiveNode, heap: Heap): void {
  deleteFromHeap(element, heap);

  let newHeight = element._height;

  // Find maximum dependency height
  for (let depLink = element._deps; depLink; depLink = depLink._nextDep) {
    const dependency = depLink._dep;
    // Unwrap firewall if present
    const actualDep = "_owner" in dependency ? dependency._owner! : dependency;

    if ("_fn" in actualDep) {
      // It's a computed node
      if (actualDep._height >= newHeight) {
        newHeight = actualDep._height + 1;
      }
    }
  }

  // If height changed, propagate to subscribers
  if (element._height !== newHeight) {
    element._height = newHeight;

    for (
      let subLink = element._subs;
      subLink !== null;
      subLink = subLink._nextSub
    ) {
      insertIntoHeapHeight(subLink._sub, heap);
    }
  }
}

// ============================================================================
// DEPENDENCY GRAPH (Linking Dependencies and Subscribers)
// ============================================================================

/**
 * Create a dependency link between observable and subscriber
 */
function link(dependency: ReactiveNode, subscriber: ReactiveNode): void {
  const previousDep = subscriber._depsTail;

  // Already linked as most recent dependency
  if (previousDep !== null && previousDep._dep === dependency) {
    return;
  }

  let nextDep: DependencyLink | null = null;
  const isRecomputing = subscriber._flags & ReactiveFlags.RecomputingDeps;

  if (isRecomputing) {
    // During recompute, check if this dependency already exists
    nextDep = previousDep !== null ? previousDep._nextDep : subscriber._deps;

    if (nextDep !== null && nextDep._dep === dependency) {
      // Already linked, just update tail
      subscriber._depsTail = nextDep;
      return;
    }
  }

  const previousSub = dependency._subsTail;

  // Check if already linked from subscriber side
  if (
    previousSub !== null &&
    previousSub._sub === subscriber &&
    (!isRecomputing || isValidLink(previousSub, subscriber))
  ) {
    return;
  }

  // Create new bidirectional link
  const newLink: DependencyLink = {
    _dep: dependency,
    _sub: subscriber,
    _nextDep: nextDep,
    _prevSub: previousSub,
    _nextSub: null,
  };

  // Update subscriber's dependency list
  if (previousDep !== null) {
    previousDep._nextDep = newLink;
  } else {
    subscriber._deps = newLink;
  }
  subscriber._depsTail = newLink;

  // Update dependency's subscriber list
  if (previousSub !== null) {
    previousSub._nextSub = newLink;
  } else {
    dependency._subs = newLink;
  }
  dependency._subsTail = newLink;
}

/**
 * Check if a link is still valid (in subscriber's current dependency list)
 */
function isValidLink(
  checkLink: DependencyLink,
  subscriber: ReactiveNode,
): boolean {
  const depsTail = subscriber._depsTail;

  if (depsTail !== null) {
    let currentLink = subscriber._deps;

    while (currentLink !== null) {
      if (currentLink === checkLink) {
        return true;
      }
      if (currentLink === depsTail) {
        break;
      }
      currentLink = currentLink._nextDep;
    }
  }

  return false;
}

/**
 * Remove a dependency link and return the next one
 */
function unlinkSubs(link: DependencyLink): DependencyLink | null {
  const dependency = link._dep;
  const nextDep = link._nextDep;
  const nextSub = link._nextSub;
  const prevSub = link._prevSub;

  // Remove from dependency's subscriber list
  if (nextSub !== null) {
    nextSub._prevSub = prevSub;
  } else {
    dependency._subsTail = prevSub;
  }

  if (prevSub !== null) {
    prevSub._nextSub = nextSub;
  } else {
    dependency._subs = nextSub;
  }

  return nextDep;
}

// ============================================================================
// SCHEDULER & QUEUE (Effect Execution)
// ============================================================================

/**
 * Queue for managing effect execution and parent/child scopes
 */
class Queue {
  _parent: Queue | null = null;
  _running = false;
  _queues: [Function[], Function[]] = [[], []]; // [render effects, user effects]
  _children: Queue[] = [];
  _pendingNodes: ReactiveNode[] = [];
  created = clock;

  static _update: (node: ReactiveNode, create?: boolean) => void;
  static _dispose: (node: ReactiveNode, zombie?: boolean) => void;

  /**
   * Add effect to queue
   */
  enqueue(type: EffectType, fn: Function): void {
    if (type) {
      this._queues[type - 1].push(fn);
    }
    schedule();
  }

  /**
   * Run all effects of given type
   */
  run(type: EffectType): void {
    if (this._queues[type - 1].length) {
      const effects = this._queues[type - 1];
      this._queues[type - 1] = [];
      runQueue(effects, type);
    }

    // Recursively run children
    for (let i = 0; i < this._children.length; i++) {
      this._children[i].run(type);
    }
  }

  /**
   * Process all pending updates
   */
  flush(): void {
    if (this._running) return;
    this._running = true;

    try {
      // Run dirty heap (recompute all dirty nodes)
      runHeap(dirtyQueue, Queue._update);

      if (activeTransition) {
        // We're in an async transition
        if (!transitionComplete(activeTransition)) {
          // Transition not complete, suspend it
          runHeap(pendingQueue, Queue._update);
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

        // Transition complete, merge back to global
        globalQueue._pendingNodes.push(...activeTransition.pendingNodes);
        globalQueue._queues[0].push(...activeTransition.queues[0]);
        globalQueue._queues[1].push(...activeTransition.queues[1]);
        transitions.delete(activeTransition);
        activeTransition = null;

        if (runPending(globalQueue._pendingNodes, false)) {
          runHeap(dirtyQueue, Queue._update);
        }
      } else if (transitions.size) {
        // Other transitions exist, run pending queue
        runHeap(pendingQueue, Queue._update);
      }

      // Commit all pending values
      for (let i = 0; i < globalQueue._pendingNodes.length; i++) {
        const node = globalQueue._pendingNodes[i];

        if (node._pendingValue !== NOT_PENDING) {
          node._value = node._pendingValue;
          node._pendingValue = NOT_PENDING;
        }

        if (node._fn) {
          Queue._dispose(node, true);
        }
      }

      globalQueue._pendingNodes.length = 0;
      clock++;
      scheduled = false;

      // Run effects
      this.run(EffectType.Render);
      this.run(EffectType.User);
    } finally {
      this._running = false;

      if (unobserved.length) {
        notifyUnobserved();
      }
    }
  }

  /**
   * Add child queue
   */
  addChild(child: Queue): void {
    this._children.push(child);
    child._parent = this;
  }

  /**
   * Remove child queue
   */
  removeChild(child: Queue): void {
    const index = this._children.indexOf(child);
    if (index >= 0) {
      this._children.splice(index, 1);
      child._parent = null;
    }
  }

  /**
   * Notify queue of status changes (pending/error)
   */
  notify(node: ReactiveNode, mask: number, flags: StatusFlags): boolean {
    if (mask & StatusFlags.Pending) {
      if (flags & StatusFlags.Pending) {
        // Track async node in transition
        if (
          activeTransition &&
          !activeTransition.asyncNodes.includes(node._error.cause)
        ) {
          activeTransition.asyncNodes.push(node._error.cause);
        }
      }
      return true;
    }

    if (this._parent) {
      return this._parent.notify(node, mask, flags);
    }

    return false;
  }

  /**
   * Initialize or update transition for async operations
   */
  initTransition(node: ReactiveNode): void {
    if (activeTransition && activeTransition.time === clock) {
      return;
    }

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

    // Move pending nodes to transition
    for (let i = 0; i < globalQueue._pendingNodes.length; i++) {
      const n = globalQueue._pendingNodes[i];
      n._transition = activeTransition;
      activeTransition.pendingNodes.push(n);
    }

    globalQueue._pendingNodes = activeTransition.pendingNodes;
  }
}

const globalQueue = new Queue();

/**
 * Schedule a flush
 */
function schedule(): void {
  if (scheduled) return;
  scheduled = true;

  if (!globalQueue._running) {
    queueMicrotask(flush);
  }
}

/**
 * Flush with infinite loop detection
 */
function flush(): void {
  let count = 0;

  while (scheduled) {
    if (++count === 100000) {
      throw new Error("Potential Infinite Loop Detected.");
    }
    globalQueue.flush();
  }
}

/**
 * Run queue of effects
 */
function runQueue(queue: Function[], type: EffectType): void {
  for (let i = 0; i < queue.length; i++) {
    queue[i](type);
  }
}

/**
 * Update pending checks for all nodes in transition
 */
function runPending(pendingNodes: ReactiveNode[], value: boolean): boolean {
  let needsReset = false;
  const nodesCopy = pendingNodes.slice();

  for (let i = 0; i < nodesCopy.length; i++) {
    const node = nodesCopy[i];
    node._transition = activeTransition;

    if (node._pendingCheck) {
      node._pendingCheck._set!(value);
      needsReset = true;
    }
  }

  return needsReset;
}

/**
 * Check if async transition is complete
 */
function transitionComplete(transition: Transition): boolean {
  let done = true;

  for (let i = 0; i < transition.asyncNodes.length; i++) {
    if (transition.asyncNodes[i]._statusFlags & StatusFlags.Pending) {
      done = false;
      break;
    }
  }

  return done;
}

/**
 * Notify signals that lost all observers
 */
function notifyUnobserved(): void {
  for (let i = 0; i < unobserved.length; i++) {
    const source = unobserved[i];
    if (!source._subs) {
      source._unobserved?.();
    }
  }
  unobserved = [];
}

// ============================================================================
// CORE REACTIVITY (Recomputation, Reading, Writing)
// ============================================================================

// Set update functions on Queue
Queue._update = recompute;
Queue._dispose = disposeChildren;

/**
 * Recompute a node's value
 */
function recompute(element: ReactiveNode, create = false): void {
  // Determine which heap to use based on zombie state
  const heap =
    element._flags & ReactiveFlags.Zombie ? pendingQueue : dirtyQueue;
  deleteFromHeap(element, heap);

  // Handle pending disposal
  if (
    element._pendingValue !== NOT_PENDING ||
    element._pendingFirstChild ||
    element._pendingDisposal
  ) {
    disposeChildren(element);
  } else {
    markDisposal(element);
    globalQueue._pendingNodes.push(element);
    element._pendingDisposal = element._disposal;
    element._pendingFirstChild = element._firstChild;
    element._disposal = null;
    element._firstChild = null;
  }

  // Set up context for computation
  const oldContext = context;
  context = element;
  element._depsTail = null;
  element._flags = ReactiveFlags.RecomputingDeps;

  let value =
    element._pendingValue === NOT_PENDING
      ? element._value
      : element._pendingValue;
  const oldHeight = element._height;
  element._time = clock;

  const prevStatusFlags = element._statusFlags;
  const prevError = element._error;
  clearStatusFlags(element);

  // Execute computation function
  try {
    value = element._fn!(value);
  } catch (e) {
    if (e instanceof NotReadyError) {
      // Async operation - mark as pending
      setStatusFlags(
        element,
        (prevStatusFlags & ~StatusFlags.Error) | StatusFlags.Pending,
        e,
      );
    } else {
      // Other error
      setError(element, e);
    }
  }

  // Notify queue if needed (for effects)
  element._notifyQueue?.();

  element._flags = ReactiveFlags.None;
  context = oldContext;

  // Clean up old dependencies
  const depsTail = element._depsTail;
  let toRemove = depsTail !== null ? depsTail._nextDep : element._deps;

  while (toRemove !== null) {
    toRemove = unlinkSubs(toRemove);
  }

  if (depsTail !== null) {
    depsTail._nextDep = null;
  } else {
    element._deps = null;
  }

  // Check if value or status changed
  const valueChanged =
    !element._equals ||
    !element._equals(
      element._pendingValue === NOT_PENDING
        ? element._value
        : element._pendingValue,
      value,
    );

  const statusFlagsChanged =
    element._statusFlags !== prevStatusFlags || element._error !== prevError;

  if (valueChanged || statusFlagsChanged) {
    if (valueChanged) {
      // Update value
      if (create || element._optimistic || element._type) {
        element._value = value;
      } else {
        if (element._pendingValue === NOT_PENDING) {
          globalQueue._pendingNodes.push(element);
        }
        element._pendingValue = value;
      }

      // Update optimistic signal if exists
      if (element._pendingSignal) {
        element._pendingSignal._set!(value);
      }
    }

    // Mark subscribers as dirty
    for (
      let subLink = element._subs;
      subLink !== null;
      subLink = subLink._nextSub
    ) {
      const subHeap =
        subLink._sub._flags & ReactiveFlags.Zombie ? pendingQueue : dirtyQueue;
      insertIntoHeap(subLink._sub, subHeap);
    }
  } else if (element._height !== oldHeight) {
    // Height changed but value didn't
    for (
      let subLink = element._subs;
      subLink !== null;
      subLink = subLink._nextSub
    ) {
      const subHeap =
        subLink._sub._flags & ReactiveFlags.Zombie ? pendingQueue : dirtyQueue;
      insertIntoHeapHeight(subLink._sub, subHeap);
    }
  }
}

/**
 * Update node if it's dirty or needs checking
 */
function updateIfNecessary(element: ReactiveNode): void {
  if (element._flags & ReactiveFlags.Check) {
    // Check all dependencies
    for (let depLink = element._deps; depLink; depLink = depLink._nextDep) {
      const dependency = depLink._dep;
      const actualDep =
        "_owner" in dependency ? dependency._owner! : dependency;

      if ("_fn" in actualDep) {
        updateIfNecessary(actualDep);
      }

      if (element._flags & ReactiveFlags.Dirty) {
        break; // Already dirty, no need to check more
      }
    }
  }

  if (element._flags & ReactiveFlags.Dirty) {
    recompute(element);
  }

  element._flags = ReactiveFlags.None;
}

/**
 * Read a signal or computed value
 */
function read(element: ReactiveNode): any {
  let currentContext = context;

  // Unwrap root context
  if (currentContext?._root) {
    currentContext = currentContext._parentComputed;
  }

  // Track dependency if we're in a reactive context
  if (currentContext && tracking) {
    link(element, currentContext);

    const owner = "_owner" in element ? element._owner! : element;

    if ("_fn" in owner) {
      // It's a computed - may need updating
      const isZombie = element._flags & ReactiveFlags.Zombie;
      const minHeight = isZombie ? pendingQueue._min : dirtyQueue._min;

      if (owner._height >= minHeight) {
        markNode(currentContext);
        markHeap(isZombie ? pendingQueue : dirtyQueue);
        updateIfNecessary(owner);
      }

      // Update subscriber height
      const height = owner._height;
      if (height >= currentContext._height) {
        currentContext._height = height + 1;
      }
    }
  }

  // Handle pending check (for isPending())
  if (pendingCheck) {
    if (!element._pendingCheck) {
      element._pendingCheck = signal(
        (element._statusFlags & StatusFlags.Pending) !== 0 ||
          !!element._transition ||
          false,
      );
      element._pendingCheck._optimistic = true;
      element._pendingCheck._set = (v) => setSignal(element._pendingCheck!, v);
    }

    const prev = pendingCheck;
    pendingCheck = null;
    prev._value = read(element._pendingCheck) || prev._value;
    pendingCheck = prev;
  }

  // Handle pending value check (for pending())
  if (pendingValueCheck) {
    if (!element._pendingSignal) {
      element._pendingSignal = signal(
        element._pendingValue === NOT_PENDING
          ? element._value
          : element._pendingValue,
      );
      element._pendingSignal._optimistic = true;
      element._pendingSignal._set = (v) =>
        queueMicrotask(() =>
          queueMicrotask(() => setSignal(element._pendingSignal!, v)),
        );
    }

    pendingValueCheck = false;
    try {
      return read(element._pendingSignal);
    } finally {
      pendingValueCheck = true;
    }
  }

  // Handle pending/error states
  if (element._statusFlags & StatusFlags.Pending) {
    if (
      (currentContext && !stale) ||
      element._statusFlags & StatusFlags.Uninitialized
    ) {
      throw element._error;
    } else if (currentContext && stale && !pendingCheck) {
      setStatusFlags(
        currentContext,
        currentContext._statusFlags | StatusFlags.Pending,
        element._error,
      );
    }
  }

  if (element._statusFlags & StatusFlags.Error) {
    if (element._time < clock) {
      // Try recomputing
      recompute(element, true);
      return read(element);
    } else {
      throw element._error;
    }
  }

  // Return appropriate value
  return !currentContext ||
    element._pendingValue === NOT_PENDING ||
    (stale &&
      !pendingCheck &&
      element._transition &&
      activeTransition !== element._transition)
    ? element._value
    : element._pendingValue;
}

/**
 * Write a signal value
 */
function setSignal(element: ReactiveNode, newValue: any): void {
  // Handle function updates
  if (typeof newValue === "function") {
    const currentValue =
      element._pendingValue === NOT_PENDING
        ? element._value
        : element._pendingValue;
    newValue = newValue(currentValue);
  }

  // Check if value actually changed
  const currentValue =
    element._pendingValue === NOT_PENDING
      ? element._value
      : element._pendingValue;
  const valueChanged =
    !element._equals || !element._equals(currentValue, newValue);

  if (!valueChanged && !element._statusFlags) {
    return; // No change
  }

  if (valueChanged) {
    if (element._optimistic) {
      // Update immediately
      element._value = newValue;
    } else {
      // Queue for batching
      if (element._pendingValue === NOT_PENDING) {
        globalQueue._pendingNodes.push(element);
      }
      element._pendingValue = newValue;
    }

    // Update optimistic signal if exists
    if (element._pendingSignal) {
      element._pendingSignal._set!(newValue);
    }
  }

  clearStatusFlags(element);
  element._time = clock;

  // Mark subscribers as dirty
  for (let link = element._subs; link !== null; link = link._nextSub) {
    const heap =
      link._sub._flags & ReactiveFlags.Zombie ? pendingQueue : dirtyQueue;
    insertIntoHeap(link._sub, heap);
  }

  if (element._subs) {
    schedule();
  }
}

/**
 * Set status flags
 */
function setStatusFlags(
  element: ReactiveNode,
  flags: StatusFlags,
  error: any = null,
): void {
  element._statusFlags = flags;
  element._error = error;
}

/**
 * Set error state
 */
function setError(element: ReactiveNode, error: any): void {
  setStatusFlags(element, StatusFlags.Error | StatusFlags.Uninitialized, error);
}

/**
 * Clear status flags
 */
function clearStatusFlags(element: ReactiveNode): void {
  setStatusFlags(element, StatusFlags.None);
}

// ============================================================================
// DISPOSAL (Cleanup)
// ============================================================================

/**
 * Mark node's children as zombies (pending disposal)
 */
function markDisposal(element: ReactiveNode): void {
  let child = element._firstChild;

  while (child) {
    child._flags |= ReactiveFlags.Zombie;

    const inHeap = child._flags & ReactiveFlags.InHeap;
    if (inHeap) {
      // Move from dirty to pending heap
      deleteFromHeap(child, dirtyQueue);
      insertIntoHeap(child, pendingQueue);
    }

    markDisposal(child);
    child = child._nextSibling;
  }
}

/**
 * Dispose node's children
 */
function disposeChildren(node: ReactiveNode, zombie?: boolean): void {
  let child = zombie ? node._pendingFirstChild : node._firstChild;

  while (child) {
    const nextChild = child._nextSibling;

    if (child._deps) {
      const childNode = child;
      const heap =
        childNode._flags & ReactiveFlags.Zombie ? pendingQueue : dirtyQueue;
      deleteFromHeap(childNode, heap);

      // Remove all dependencies
      let toRemove = childNode._deps;
      while (toRemove !== null) {
        toRemove = unlinkSubs(toRemove);
      }

      childNode._deps = null;
      childNode._depsTail = null;
      childNode._flags = ReactiveFlags.None;
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

/**
 * Run disposal functions
 */
function runDisposal(node: ReactiveNode, zombie?: boolean): void {
  const disposal = zombie ? node._pendingDisposal : node._disposal;

  if (!disposal || disposal === NOT_PENDING) {
    return;
  }

  if (Array.isArray(disposal)) {
    for (let i = 0; i < disposal.length; i++) {
      disposal[i]();
    }
  } else {
    disposal();
  }

  if (zombie) {
    node._pendingDisposal = null;
  } else {
    node._disposal = null;
  }
}

// ============================================================================
// NODE CREATION (Signals, Computed, Effects)
// ============================================================================

/**
 * Apply options to a node
 */
function withOptions(obj: ReactiveNode, options: any): ReactiveNode {
  obj._name = options?.name ?? (obj._fn ? "computed" : "signal");
  obj._id = options?.id;
  obj._equals = options?.equals !== undefined ? options.equals : isEqual;
  obj._pureWrite = !!options?.pureWrite;
  obj._unobserved = options?.unobserved;

  if (options?._internal) {
    Object.assign(obj, options._internal);
  }

  return obj;
}

/**
 * Generate child ID
 */
function getNextChildId(owner: ReactiveNode): string {
  if (owner._id != null) {
    return formatId(owner._id, owner._childCount!++);
  }
  throw new Error("Cannot get child id from owner without an id");
}

/**
 * Format ID in base-36
 */
function formatId(prefix: string, id: number): string {
  const num = id.toString(36);
  const len = num.length - 1;
  return prefix + (len ? String.fromCharCode(64 + len) : "") + num;
}

/**
 * Default equality check
 */
function isEqual(a: any, b: any): boolean {
  return a === b;
}

/**
 * Create a computed value
 */
function computed(
  fn: (prev: any) => any,
  initialValue: any,
  options?: any,
): ReactiveNode {
  const self: ReactiveNode = withOptions(
    {
      _disposal: null,
      _queue: globalQueue,
      _context: defaultContext,
      _childCount: 0,
      _fn: fn,
      _value: initialValue,
      _height: 0,
      _child: null,
      _nextHeap: undefined,
      _prevHeap: null,
      _deps: null,
      _depsTail: null,
      _subs: null,
      _subsTail: null,
      _parent: context,
      _nextSibling: null,
      _firstChild: null,
      _flags: ReactiveFlags.None,
      _statusFlags: StatusFlags.Uninitialized,
      _time: clock,
      _pendingValue: NOT_PENDING,
      _pendingDisposal: null,
      _pendingFirstChild: null,
    } as ReactiveNode,
    options,
  );

  self._prevHeap = self;

  const parent = context?._root ? context._parentComputed : context;

  if (context) {
    // Inherit queue and context from parent
    if (context._queue) self._queue = context._queue;
    if (context._context) self._context = context._context;

    // Add to parent's child list
    const lastChild = context._firstChild;
    if (lastChild === null) {
      context._firstChild = self;
    } else {
      self._nextSibling = lastChild;
      context._firstChild = self;
    }

    if (parent) {
      if (parent._depsTail === null || options?._forceRun) {
        // No dependencies yet or forced run - compute immediately
        self._height = parent._height;
        recompute(self, true);
      } else {
        // Has dependencies - queue for later
        self._height = parent._height + 1;
        insertIntoHeap(self, dirtyQueue);
      }
    }
  } else {
    // No parent - compute immediately
    recompute(self, true);
  }

  return self;
}

/**
 * Create an async computed value (handles Promises and AsyncIterators)
 */
function asyncComputed(
  asyncFn: (prev: any, refreshing: boolean) => any,
  initialValue: any,
  options?: any,
): ReactiveNode {
  let lastResult: any = undefined;
  let refreshing = false;

  const fn = (prev: any) => {
    const result = asyncFn(prev, refreshing);
    refreshing = false;
    lastResult = result;

    const isPromise = result instanceof Promise;
    const iterator = result[Symbol.asyncIterator];

    if (!isPromise && !iterator) {
      // Sync result
      return result;
    }

    if (isPromise) {
      // Handle Promise
      result
        .then((v: any) => {
          if (lastResult !== result) return; // Stale
          globalQueue.initTransition(self);
          setSignal(self, v);
          flush();
        })
        .catch((e: any) => {
          if (lastResult !== result) return; // Stale
          globalQueue.initTransition(self);
          setError(self, e);
          flush();
        });
    } else {
      // Handle AsyncIterator
      (async () => {
        try {
          for await (let value of result) {
            if (lastResult !== result) return; // Stale
            globalQueue.initTransition(self);
            setSignal(self, value);
            flush();
          }
        } catch (error) {
          if (lastResult !== result) return; // Stale
          globalQueue.initTransition(self);
          setError(self, error);
          flush();
        }
      })();
    }

    // Throw NotReadyError to suspend
    globalQueue.initTransition(context!);
    throw new NotReadyError(context);
  };

  const self = computed(fn, initialValue, options);

  // Add refresh method
  self._refresh = () => {
    refreshing = true;
    recompute(self);
    flush();
  };

  return self;
}

/**
 * Create a signal
 *
 * @param value - Initial value
 * @param options - Signal options
 * @param firewall - Optional firewall node for nested signals
 */
function signal(
  value: any,
  options?: any,
  firewall: ReactiveNode | null = null,
): ReactiveNode {
  if (firewall !== null) {
    // Create nested signal behind firewall
    return (firewall._child = withOptions(
      {
        _value: value,
        _subs: null,
        _subsTail: null,
        _owner: firewall,
        _nextChild: firewall._child,
        _statusFlags: StatusFlags.None,
        _time: clock,
        _pendingValue: NOT_PENDING,
      } as ReactiveNode,
      options,
    ));
  } else {
    // Create regular signal
    return withOptions(
      {
        _value: value,
        _subs: null,
        _subsTail: null,
        _statusFlags: StatusFlags.None,
        _time: clock,
        _pendingValue: NOT_PENDING,
      } as ReactiveNode,
      options,
    );
  }
}

// ============================================================================
// UTILITIES
// ============================================================================

/**
 * Execute function without tracking dependencies
 */
function untrack<T>(fn: () => T): T {
  if (!tracking) return fn();

  tracking = false;
  try {
    return fn();
  } finally {
    tracking = true;
  }
}

/**
 * Get current reactive observer
 */
function getObserver(): ReactiveNode | null {
  return tracking ? context : null;
}

/**
 * Get current owner (reactive context)
 */
function getOwner(): ReactiveNode | null {
  return context;
}

/**
 * Register cleanup function
 */
function onCleanup(fn: Function): Function {
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

/**
 * Create a new reactive root
 */
function createRoot(
  init: ((dispose: () => void) => void) | (() => void),
  options?: any,
): any {
  const parent = context;

  const owner: ReactiveNode = {
    _root: true,
    _parentComputed: parent?._root ? parent._parentComputed : parent,
    _disposal: null,
    _id: options?.id ?? (parent?._id ? getNextChildId(parent) : undefined),
    _queue: parent?._queue ?? globalQueue,
    _context: parent?._context || defaultContext,
    _childCount: 0,
    _pendingDisposal: null,
    _pendingFirstChild: null,
    _parent: parent,
  } as ReactiveNode;

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
    !init.length
      ? (init as () => void)
      : () =>
          (init as (dispose: () => void) => void)(() => disposeChildren(owner)),
  );
}

/**
 * Run function with specific owner context
 */
function runWithOwner(owner: ReactiveNode, fn: () => any): any {
  const oldContext = context;
  context = owner;

  try {
    return fn();
  } finally {
    context = oldContext;
  }
}

/**
 * Read stale (current) values instead of pending
 */
function staleValues<T>(fn: () => T, set = true): T {
  const prevStale = stale;
  stale = set;

  try {
    return fn();
  } finally {
    stale = prevStale;
  }
}

/**
 * Read pending (optimistic) values
 */
function pending<T>(fn: () => T): T {
  const prevLatest = pendingValueCheck;
  pendingValueCheck = true;

  try {
    return staleValues(fn, false);
  } finally {
    pendingValueCheck = prevLatest;
  }
}

/**
 * Check if any signals are pending
 */
function isPending(fn: () => void, loadingValue?: boolean): boolean {
  const current = pendingCheck;
  pendingCheck = { _value: false };

  try {
    staleValues(fn);
    return pendingCheck._value;
  } catch (err) {
    if (!(err instanceof NotReadyError)) return false;
    if (loadingValue !== undefined) return loadingValue;
    throw err;
  } finally {
    pendingCheck = current;
  }
}

// ============================================================================
// CONTEXT API
// ============================================================================

interface ContextObject<T = any> {
  id: symbol;
  defaultValue?: T;
}

/**
 * Create a context
 */
function createContext<T = any>(
  defaultValue?: T,
  description?: string,
): ContextObject<T> {
  return {
    id: Symbol(description),
    defaultValue,
  };
}

/**
 * Get context value
 */
function getContext<T>(contextObj: ContextObject<T>, owner = getOwner()): T {
  if (!owner) {
    throw new NoOwnerError();
  }

  const value = hasContext(contextObj, owner)
    ? owner._context![contextObj.id]
    : contextObj.defaultValue;

  if (isUndefined(value)) {
    throw new ContextNotFoundError();
  }

  return value;
}

/**
 * Set context value
 */
function setContext<T>(
  contextObj: ContextObject<T>,
  value: T,
  owner = getOwner(),
): void {
  if (!owner) {
    throw new NoOwnerError();
  }

  owner._context = {
    ...owner._context,
    [contextObj.id]: isUndefined(value) ? contextObj.defaultValue : value,
  };
}

/**
 * Check if context exists
 */
function hasContext<T>(
  contextObj: ContextObject<T>,
  owner: ReactiveNode | null,
): boolean {
  return !isUndefined(owner?._context?.[contextObj.id]);
}

/**
 * Check if value is undefined
 */
function isUndefined(value: any): boolean {
  return typeof value === "undefined";
}

// ============================================================================
// EFFECTS
// ============================================================================

/**
 * Create an effect
 */
function effect(
  compute: (prev: any) => any,
  effectFn: (value: any, prev: any) => void | Function,
  errorFn: ((error: any, reset: () => void) => void) | undefined,
  initialValue: any,
  options?: any,
): void {
  let initialized = false;

  const node = computed(compute, initialValue, {
    ...options,
    _forceRun: true,
    equals(prev: any, val: any) {
      const equal = isEqual(prev, val);

      if (initialized) {
        node._modified = !equal;

        if (!equal && !(node._statusFlags & StatusFlags.Error)) {
          node._queue!.enqueue(node._type!, runEffect.bind(node));
        }
      }

      return equal;
    },
    _internal: {
      _modified: true,
      _prevValue: initialValue,
      _effectFn: effectFn,
      _errorFn: errorFn,
      _cleanup: undefined,
      _queue: getOwner()?._queue ?? globalQueue,
      _type: options?.render ? EffectType.Render : EffectType.User,
      _notifyQueue() {
        if (this._type === EffectType.Render) {
          this._queue!.notify(
            this,
            StatusFlags.Pending | StatusFlags.Error,
            this._statusFlags,
          );
        }
      },
    },
  });

  initialized = true;

  // Wrap compute for render effects to use stale values
  if (node._type === EffectType.Render) {
    const originalFn = node._fn!;
    node._fn = (p) =>
      !(node._statusFlags & StatusFlags.Error)
        ? staleValues(() => compute(p))
        : compute(p);
  }

  // Run initial effect (unless deferred)
  if (
    !options?.defer &&
    !(node._statusFlags & (StatusFlags.Error | StatusFlags.Pending))
  ) {
    if (node._type === EffectType.User) {
      node._queue!.enqueue(node._type, runEffect.bind(node));
    } else {
      runEffect.call(node);
    }
  }

  // Register cleanup
  onCleanup(() => node._cleanup?.());

  if (!node._parent) {
    console.warn(
      "Effects created outside a reactive context will never be disposed",
    );
  }
}

/**
 * Run effect function (bound to effect node as `this`)
 */
function runEffect(this: ReactiveNode): void {
  if (!this._modified) return;

  // Clean up previous effect
  this._cleanup?.();
  this._cleanup = undefined;

  try {
    this._cleanup = this._effectFn!(this._value, this._prevValue);
  } catch (error) {
    this._queue!.notify(this, StatusFlags.Pending, StatusFlags.None);

    if (this._type === EffectType.User) {
      try {
        if (this._errorFn) {
          this._errorFn(error, () => {
            this._cleanup?.();
            this._cleanup = undefined;
          });
          return;
        } else {
          console.error(error);
          return;
        }
      } catch (e) {
        error = e;
      }
    }

    if (!this._queue!.notify(this, StatusFlags.Error, StatusFlags.Error)) {
      throw error;
    }
  } finally {
    this._prevValue = this._value;
    this._modified = false;
  }
}

// ============================================================================
// PUBLIC API
// ============================================================================

/**
 * Create a signal (getter/setter pair or computed)
 */
function createSignal<T>(
  initialValue: T,
  options?: any,
): [() => T, (v: T | ((prev: T) => T)) => void];
function createSignal<T>(
  computeFn: (prev: T) => T,
  initialValue: T,
  options?: any,
): [() => T, (v: T | ((prev: T) => T)) => void];
function createSignal(first: any, second?: any, third?: any): any {
  if (typeof first === "function") {
    // Computed signal
    const node = computed(first, second, third);
    return [read.bind(null, node), setSignal.bind(null, node)];
  }

  // Regular signal
  const owner = getOwner();
  const needsId = owner?._id != null;
  const node = signal(
    first,
    needsId ? { id: getNextChildId(owner!), ...second } : second,
  );

  return [read.bind(null, node), setSignal.bind(null, node)];
}

/**
 * Create a memo (computed value)
 */
function createMemo<T>(
  compute: (prev?: T) => T,
  value?: T,
  options?: any,
): () => T {
  const node = computed(compute, value, options);
  return read.bind(null, node);
}

/**
 * Create an async computed
 */
function createAsync<T>(
  compute: (prev: T, refreshing: boolean) => any,
  value?: T,
  options?: any,
): (() => T) & { refresh: () => void } {
  const node = asyncComputed(compute, value, options);
  const ret = read.bind(null, node) as any;
  ret.refresh = node._refresh;
  return ret;
}

/**
 * Create an effect
 */
function createEffect<T>(
  compute: () => T,
  effectFn: ((value: T, prev: T) => void | Function) & {
    effect?: any;
    error?: any;
  },
  value?: T,
  options?: any,
): void {
  effect(compute, effectFn.effect || effectFn, effectFn.error, value, {
    ...options,
    name: options?.name ?? "effect",
  });
}

/**
 * Create a render effect
 */
function createRenderEffect<T>(
  compute: () => T,
  effectFn: (value: T, prev: T) => void | Function,
  value?: T,
  options?: any,
): void {
  effect(compute, effectFn, undefined, value, {
    render: true,
    ...{ ...options, name: options?.name ?? "effect" },
  });
}

/**
 * Create a tracked effect (not implemented)
 */
function createTrackedEffect(compute: any, options?: any): void {
  // Not implemented
}

/**
 * Create a reaction (not implemented)
 */
function createReaction(effectFn: any, options?: any): void {
  // Not implemented
}

/**
 * Resolve async signal to a Promise
 */
function resolve<T>(fn: () => T): Promise<T> {
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

/**
 * Create optimistic signal (not implemented)
 */
function createOptimistic(first: any, second?: any, third?: any): any {
  return {};
}

// ============================================================================
// EXPORTS
// ============================================================================

export {
  // Errors
  NotReadyError,
  NoOwnerError,
  ContextNotFoundError,

  // Core
  Queue,
  SUPPORTS_PROXY,

  // Signals
  createSignal,
  createMemo,
  createAsync,
  createOptimistic,

  // Effects
  createEffect,
  createRenderEffect,
  createTrackedEffect,
  createReaction,

  // Context
  createContext,
  getContext,
  setContext,

  // Utilities
  untrack,
  getObserver,
  getOwner,
  onCleanup,
  isEqual,

  // Async
  pending,
  isPending,
  resolve,

  // Scheduler
  flush,
};
