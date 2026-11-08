"use strict";
(function () {
  // Lootnest's window never opens other windows (links open in the browser).
  try { window.open = () => null; } catch (e) { /* read-only */ }
