import {
  createSignal,
  createAsync,
  createRenderEffect,
  isPending,
} from "./reactivity.ts";

const states = createAsync(() => getStates());
const [state, setState] = createSignal(() => states()[0]);
const cities = createAsync(() => getCities(state()));
const [city, setCity] = createSignal(() => cities()[0]);

app.appendChild(
  Select({
    value: state,
    options: states,
    onChange: setState,
  }),
);
app.appendChild(
  Select({
    value: city,
    options: cities,
    onChange: setCity,
  }),
);

const div = app.appendChild(document.createElement("div"));
createRenderEffect(
  () => [city(), state()],
  ([c, s]) => {
    div.textContent = `Selection: ${c}, ${s}`;
  },
);

function Select({ value, onChange, options }) {
  const elm = document.createElement("div");

  const select = elm.appendChild(document.createElement("select"));
  const defaultOption = document.createElement("option");
  defaultOption.textContent = "Loading...";
  select.appendChild(defaultOption);
  select.onchange = (e) => onChange(e.target.value);

  createRenderEffect(value, (v) => {
    select.value = v;
  });

  createRenderEffect(
    () => isPending(options, true),
    (p) => {
      select.style.opacity = p ? "0.5" : "1";
    },
  );

  createRenderEffect(options, (os) => {
    select.innerHTML = os.map((o) => `<option value="${o}">${o}`).join("");
  });

  return elm;
}

const stateCitites: Record<string, string[]> = {
  Utah: ["Salt Lake City", "Provo", "West Valley City"],
  California: ["Los Angeles", "San Francisco", "San Diego"],
  Texas: ["Houston", "Dallas", "Austin"],
  Florida: ["Miami", "Orlando", "Tampa"],
  "New York": ["New York City", "Buffalo", "Rochester"],
};

async function getCities(state: string) {
  await new Promise((res) => setTimeout(res, 500));
  return stateCitites[state] ?? [];
}

async function getStates() {
  await new Promise((res) => setTimeout(res, 500));
  return ["California", "Florida", "New York", "Texas", "Utah"];
}
