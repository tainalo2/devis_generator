/* @ds-bundle: {"format":4,"namespace":"DevisUI","components":[{"name":"Button"},{"name":"Badge"},{"name":"Input"},{"name":"Label"},{"name":"Separator"},{"name":"Spinner"}]} */
/* Devis Generator : VIBED — hand-written from frontend/src/components/ui/*.tsx (tainalo2/devis_generator@vibed bf7cd0b).
   Same variants, sizes and markup as the source; classes map to components/bundle.css. */
(function () {
  "use strict";
  var React = window.React;
  var h = React.createElement;

  function cx() {
    var out = [];
    for (var i = 0; i < arguments.length; i++) {
      var a = arguments[i];
      if (a) out.push(a);
    }
    return out.join(" ");
  }
  function omit(props, keys) {
    var o = {};
    for (var k in props) {
      if (Object.prototype.hasOwnProperty.call(props, k) && keys.indexOf(k) === -1) o[k] = props[k];
    }
    return o;
  }

  var BUTTON_VARIANTS = ["default", "outline", "secondary", "ghost", "destructive", "link"];
  var BUTTON_SIZES = ["default", "xs", "sm", "lg", "icon", "icon-xs", "icon-sm", "icon-lg"];

  var Button = React.forwardRef(function Button(props, ref) {
    var variant = BUTTON_VARIANTS.indexOf(props.variant) > -1 ? props.variant : "default";
    var size = BUTTON_SIZES.indexOf(props.size) > -1 ? props.size : "default";
    var rest = omit(props, ["variant", "size", "className"]);
    return h("button", Object.assign({ type: "button" }, rest, {
      ref: ref,
      "data-slot": "button",
      className: cx("dg-button", "dg-button--" + variant, "dg-button--size-" + size, props.className)
    }));
  });

  var BADGE_VARIANTS = ["default", "secondary", "outline", "success", "warning", "muted"];
  function Badge(props) {
    var variant = BADGE_VARIANTS.indexOf(props.variant) > -1 ? props.variant : "default";
    var rest = omit(props, ["variant", "className"]);
    return h("span", Object.assign({}, rest, { className: cx("dg-badge", "dg-badge--" + variant, props.className) }));
  }

  var Input = React.forwardRef(function Input(props, ref) {
    var rest = omit(props, ["className", "type"]);
    return h("input", Object.assign({}, rest, {
      ref: ref,
      type: props.type || "text",
      className: cx("dg-input", props.className)
    }));
  });

  function Label(props) {
    var rest = omit(props, ["className"]);
    return h("label", Object.assign({}, rest, { className: cx("dg-label", props.className) }));
  }

  function Separator(props) {
    var orientation = props.orientation === "vertical" ? "vertical" : "horizontal";
    var rest = omit(props, ["className", "orientation"]);
    return h("div", Object.assign({ role: "separator", "aria-orientation": orientation }, rest, {
      "data-slot": "separator",
      "data-orientation": orientation,
      className: cx("dg-separator", props.className)
    }));
  }

  function Spinner(props) {
    var label = props.label == null ? "Chargement" : props.label;
    return h("span", { role: "status", "aria-live": "polite", className: cx("dg-spinner", props.className) },
      h("span", { className: "dg-spinner__ring", "aria-hidden": true }),
      h("span", { className: "dg-sr-only" }, label));
  }

  window.DevisUI = {
    Button: Button,
    Badge: Badge,
    Input: Input,
    Label: Label,
    Separator: Separator,
    Spinner: Spinner
  };
})();
