import {
  signal,
  computed,
  read,
  setSignal,
  asyncComputed,
  latest,
  stabilize,
} from "./reactivity.ts";

function fetch(value: number) {
  console.log("Fetch data", "D" + value);
  return new Promise((res) => setTimeout(() => res("D" + value), 2000));
}
const count = signal(0);
const selection = signal(0);
const data = asyncComputed(() => fetch(read(selection)));
computed(() => {
  const d = read(data);
  console.log("Render data", d);
  computed(() => {
    latest(() => console.log("S" + read(selection), read(count)));
  });
});
stabilize();

setInterval(() => {
  setSignal(count, read(count) + 1);
  stabilize();
}, 500);

setInterval(() => {
  setSignal(selection, read(selection) + 1);
  stabilize();
}, 5000);
