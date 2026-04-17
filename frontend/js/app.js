import { getApiBase } from "./config.js";
import { setSession, clearSession, getSession } from "./auth.js";
import {
  registerUser,
  listProducts,
  getProfile,
  createOrder,
  listOrders,
} from "./api.js";

function $(id) {
  return document.getElementById(id);
}

function setStatus(el, text, ok = true) {
  if (!el) return;
  el.textContent = text;
  el.classList.toggle("is-error", !ok);
}

function renderProducts(data, container) {
  if (!container) return;
  const products = data?.products || [];
  if (!products.length) {
    container.innerHTML = "<p class=\"muted\">No products yet.</p>";
    return;
  }
  const rows = products
    .map(
      (p) =>
        `<li><strong>${escapeHtml(p.name || "")}</strong> · ${escapeHtml(p.category || "")} · $${Number(p.price).toFixed(2)} · ${p.stock} available <span class="muted">#${p.productId}</span></li>`
    )
    .join("");
  container.innerHTML = `<ul class="product-list">${rows}</ul>`;
}

function renderOrders(data, container) {
  if (!container) return;
  const orders = data?.orders || [];
  if (!orders.length) {
    container.innerHTML = "<p class=\"muted\">No orders yet.</p>";
    return;
  }
  const rows = orders
    .map(
      (o) =>
        `<li>Order <strong>#${o.orderId}</strong> · ${escapeHtml(o.status || "—")} · $${Number(o.totalAmount).toFixed(2)}</li>`
    )
    .join("");
  container.innerHTML = `<ul class="product-list">${rows}</ul>`;
}

function escapeHtml(s) {
  return String(s)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

async function onRegister(e) {
  e.preventDefault();
  const status = $("status-register");
  const email = $("reg-email")?.value?.trim();
  const password = $("reg-password")?.value || "";
  const name = $("reg-name")?.value?.trim();
  try {
    await registerUser({ email, password, name });
    setStatus(status, "Welcome! Sign in below with the email and password you chose.", true);
  } catch (err) {
    setStatus(status, err.message || String(err), false);
  }
}

async function onSaveSession(e) {
  e.preventDefault();
  const status = $("status-session");
  const email = $("sess-email")?.value?.trim();
  const password = $("sess-password")?.value || "";
  setSession(email, password);
  setStatus(status, "Signed in on this browser for checkout and your details.", true);
}

function onClearSession() {
  clearSession();
  $("sess-email").value = "";
  $("sess-password").value = "";
  setStatus($("status-session"), "Signed out on this browser.", true);
}

async function onLoadProducts() {
  const status = $("status-catalog");
  const out = $("products-out");
  setStatus(status, "Loading…", true);
  try {
    const data = await listProducts(1, 20);
    renderProducts(data, out);
    setStatus(status, `Showing ${(data.products || []).length} bouquet(s).`, true);
  } catch (err) {
    setStatus(status, err.message || String(err), false);
    out.innerHTML = "";
  }
}

function renderProfileCard(data, el) {
  if (!el) return;
  const name = data?.name ?? data?.fullName;
  const email = data?.email;
  const uid = data?.userId ?? data?.user_id;
  const bits = [];
  if (name) {
    bits.push(
      `<p><span class="profile-card__k">Name</span> ${escapeHtml(String(name))}</p>`
    );
  }
  if (email) {
    bits.push(
      `<p><span class="profile-card__k">Email</span> ${escapeHtml(String(email))}</p>`
    );
  }
  if (uid != null && uid !== "") {
    bits.push(
      `<p><span class="profile-card__k">Reference</span> ${escapeHtml(String(uid))}</p>`
    );
  }
  if (!bits.length) {
    el.innerHTML = "<p class=\"muted\">No details returned.</p>";
    return;
  }
  el.innerHTML = bits.join("");
}

async function onProfile() {
  const status = $("status-session");
  const out = $("profile-out");
  const { email } = getSession();
  if (!email) {
    setStatus(status, "Sign in above with your email and password first.", false);
    if (out) {
      out.innerHTML = "";
      out.hidden = true;
    }
    return;
  }
  try {
    const data = await getProfile();
    renderProfileCard(data, out);
    if (out) out.hidden = false;
    setStatus(status, "Here are your saved details.", true);
  } catch (err) {
    setStatus(status, err.message || String(err), false);
    if (out) {
      out.innerHTML = "";
      out.hidden = true;
    }
  }
}

async function onPlaceOrder(e) {
  e.preventDefault();
  const status = $("status-orders");
  const pid = parseInt($("order-product-id")?.value?.trim() || "0", 10) || 0;
  const qty = parseInt($("order-qty")?.value?.trim() || "0", 10) || 0;
  if (!pid || !qty) {
    setStatus(status, "Enter a bouquet number and quantity.", false);
    return;
  }
  try {
    const data = await createOrder([{ productId: pid, quantity: qty }]);
    const oid = data.orderId ?? data.order_id;
    setStatus(status, `Thanks — your order is in. Reference #${oid}.`, true);
  } catch (err) {
    setStatus(status, err.message || String(err), false);
  }
}

async function onListOrders() {
  const status = $("status-orders");
  const out = $("orders-out");
  setStatus(status, "Loading orders…", true);
  try {
    const data = await listOrders(1, 10);
    renderOrders(data, out);
    setStatus(status, `Showing your last ${(data.orders || []).length} order(s).`, true);
  } catch (err) {
    setStatus(status, err.message || String(err), false);
    out.innerHTML = "";
  }
}

function init() {
  const hint = $("api-base-hint");
  if (hint) {
    hint.textContent = getApiBase();
  }
  const s = getSession();
  if ($("sess-email")) $("sess-email").value = s.email;
  if ($("sess-password")) $("sess-password").value = s.password;

  $("form-register")?.addEventListener("submit", onRegister);
  $("form-session")?.addEventListener("submit", onSaveSession);
  $("btn-session-clear")?.addEventListener("click", onClearSession);
  $("btn-products")?.addEventListener("click", onLoadProducts);
  $("btn-profile")?.addEventListener("click", onProfile);
  $("form-order")?.addEventListener("submit", onPlaceOrder);
  $("btn-orders")?.addEventListener("click", onListOrders);
}

document.addEventListener("DOMContentLoaded", init);
