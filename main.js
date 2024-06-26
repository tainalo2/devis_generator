//code by tainalo2
window.jsPDF = window.jspdf.jsPDF
window.html2canvas = html2canvas;

const regexFirstName = new RegExp("^[a-zA-Z\u00C0-\u024F\-]+$");
const regexLastName = new RegExp("^[a-zA-Z\u00C0-\u024F\- ]+$");
const regexSiren = new RegExp("^\\d{9}$");
const regexCompanyName = new RegExp("^[\w\s'àéèù'-\u00C0-\u024F]+$");
const regexEmail = new RegExp("^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$")
const regexNumber = new RegExp("[0-9]");
const regexIntFloat = new RegExp("[0-9]*\.?[0-9]*")
const regexTwoNumbers = new RegExp("[0-9]{3}");
const regexPhoneNumber = new RegExp("^(([0-9]{2}-){4})([0-9]{2})$");
const rootRoute = null;
const originRoute = "home";
var pseudo = "";
var user_templates = "";
let signaturePad1;

function toggleLightMode(element) {
    if (element.checked) {
        document.querySelector(':root').style.setProperty('--main-bg-color', 'black');
        document.querySelector(':root').style.setProperty('--main-second-color', 'white');
        document.querySelector(':root').style.setProperty('--main-tier_color', 'rgb(53, 53, 53)');
        document.querySelector(':root').style.setProperty('--main-quater_color', 'rgb(245, 245, 245)');
        document.querySelector(':root').style.setProperty('--inactive-bg-color', 'rgb(95, 95, 95)');
        document.querySelector(':root').style.setProperty('--inactive-second-color', 'rgb(240, 240, 240)');
        document.querySelector(':root').style.setProperty('--filter-main-color-svg', 'invert(100%) sepia(2%) saturate(906%) hue-rotate(227deg) brightness(121%) contrast(100%)');
        document.querySelector(':root').style.setProperty('--filter-second-color-svg', 'invert(0%) sepia(2%) saturate(906%) hue-rotate(227deg) brightness(121%) contrast(100%)');
        document.querySelector(':root').style.setProperty('--basic_shadow', '0 1px 2px rgba(255, 255, 255, 0.11), 0 2px 4px rgba(255, 255, 255, 0.24), 0 4px 8px rgba(255, 255, 255, 0.61), 0 8px 16px rgba(255, 255, 255, 0.29), 0 16px 32px rgba(255, 255, 255, 0.29), 0 32px 64px rgba(255, 255, 255, 0.31');
        signaturePad1.penColor = "white";
    } else {
        document.querySelector(':root').style.setProperty('--main-bg-color', 'white');
        document.querySelector(':root').style.setProperty('--main-second-color', 'black');
        document.querySelector(':root').style.setProperty('--main-tier_color', 'rgb(245, 245, 245)');
        document.querySelector(':root').style.setProperty('--main-quater_color', 'rgb(53, 53, 53)');
        document.querySelector(':root').style.setProperty('--inactive-bg-color', 'rgb(240, 240, 240)');
        document.querySelector(':root').style.setProperty('--inactive-second-color', 'rgb(95, 95, 95)');
        document.querySelector(':root').style.setProperty('--filter-main-color-svg', 'invert(0%) sepia(2%) saturate(906%) hue-rotate(227deg) brightness(121%) contrast(100%)');
        document.querySelector(':root').style.setProperty('--filter-second-color-svg', 'invert(100%) sepia(2%) saturate(906%) hue-rotate(227deg) brightness(121%) contrast(100%)');
        document.querySelector(':root').style.setProperty('--basic_shadow', '0 1px 2px rgba(0, 0, 0, .07), 0 2px 4px rgba(0, 0, 0, .03), 0 4px 8px rgba(0, 0, 0, .03), 0 8px 16px rgba(0, 0, 0, .03), 0 16px 32px rgba(0, 0, 0, .03), 0 32px 64px rgba(0, 0, 0, .03)');
        signaturePad1.penColor = "black";
    }
}

function inputError(element) {
    var value = element.value.trim();
    if (element.classList.contains('input_firstName')) {
        if (value == "") {

        } else if (value != "" && regexFirstName.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.classList.contains('input_lastName')) {
        if (value == "") {

        } else if (value != "" && regexLastName.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.classList.contains('input_jecéplu')) {
        if (value != "") {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.classList.contains('input_siren')) {
        if (value == "") {

        } else if (value != "" && regexSiren.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.classList.contains('input_email')) {
        if (value == "") {

        } else if (value != "" && regexEmail.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.classList.contains("input_phone_number")) {
        if (!regexNumber.test(value.slice(-1)) || value.length > 14) {
            element.value = value.slice(0, -1);
        } else if (regexTwoNumbers.test(value.slice(-3))) {
            element.value = value.slice(0, -1) + '-' + value.slice(-1);
        }

        var value = element.value.trim();
        element = element.parentElement;
        if (value == "") {

        } else if (regexPhoneNumber.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.classList.contains('input_company_name')) {
        if (value == "") {

        } else if (value != "" && regexCompanyName.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.id == "input_indemnite") {
        value = value.replace("%", "");
        if (parseFloat(value) >= 15.21 && regexIntFloat.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    }

}

function showHideError(status, element) {
    var error_element = element.parentElement.querySelector('.error_container');
    if (status) {
        element.classList.remove("input_error");
        element.classList.add("input_valid");
        error_element.style.paddingTop = "0rem";
        error_element.style.height = "0";
    } else if (status == false) {
        element.classList.add("input_error");
        element.classList.remove("input_valid");
        error_element.style.paddingTop = "1.2rem";
        error_element.style.height = "auto";
    } else if (status == null) {
        error_element.style.paddingTop = "0rem";
        error_element.style.height = "0";
        if (element.classList.contains('input_valid')) {
            element.classList.remove("input_valid");
        }
    }
}

let placeholder;
let prevEle;
let nextEle;
let draggingEle = "";
let cursorX = 0;
let cursorY = 0;
let xDrag = 0;
let yDrag = 0;
let draggingStatus = false;

function DragOn(e) {
    if (e.target.classList.contains('svg_devis_line_dots')) {
        draggingEle = e.target.parentElement;
    } else {
        draggingEle = e.target.parentElement.parentElement.parentElement;
    }
    const rect = draggingEle.getBoundingClientRect();
    xDrag = cursorX - rect.left;
    //yDrag = e.pageY - rect.top;
    yDrag = rect.height;
    document.addEventListener('mousemove', DragMove);
    document.addEventListener('mouseup', DragOff);
}

function DragMove(e) {

    const draggingRect = draggingEle.getBoundingClientRect();


    if (!draggingStatus) {
        draggingStatus = true;
        draggingEle.style.position = 'absolute';
        draggingEle.style.zIndex = '999';
        draggingEle.style.opacity = '0.4';
        draggingEle.style.maxWidth = `${document.getElementById("first_devis_lines_container").offsetWidth}px`
        placeholder = document.createElement('div');
        placeholder.classList.add('placeholder');
        draggingEle.parentNode.insertBefore(
            placeholder,
            draggingEle.nextSibling
        );
        placeholder.style.height = `${draggingRect.height - 4}px`;
        draggingEle.parentElement.parentElement.querySelector(".devis_add_button_icon").style.display = "none";
        draggingEle.parentElement.parentElement.querySelector(".devis_trash").style.display = "block";
    }

    draggingEle.style.top = `${cursorY - yDrag}px`;
    draggingEle.style.left = `${cursorX - xDrag}px`;

    const prevEle = draggingEle.previousElementSibling;
    const nextEle = placeholder.nextElementSibling;
    if (prevEle && isAbove(draggingEle, prevEle)) {
        swap(placeholder, draggingEle);
        swap(placeholder, prevEle);
        return;
    }
    if (nextEle && isAbove(nextEle, draggingEle)) {
        swap(nextEle, placeholder);
        swap(nextEle, draggingEle);
    }

}

function DragOff(e) {
    draggingEle.style.removeProperty('position');
    draggingEle.style.removeProperty('zIndex');
    draggingEle.style.removeProperty('opacity');
    document.removeEventListener('mousemove', DragMove);
    document.removeEventListener('mouseup', DragOff);
    placeholder && placeholder.parentNode.removeChild(placeholder);
    document.querySelector(".devis_add_button_icon").style.display = "block";
    document.querySelector(".devis_trash").style.display = "none";
    if (onElement(draggingEle.parentElement.parentElement.querySelector(".devis_line_buttons_container"))) {
        draggingEle.remove();
    }
    draggingStatus = false;
}

const isAbove = function (nodeA, nodeB) {
    // Get the bounding rectangle of nodes
    const rectA = nodeA.getBoundingClientRect();
    const rectB = nodeB.getBoundingClientRect();

    return rectA.top + rectA.height / 2 < rectB.top + rectB.height / 2;
};

const swap = function (nodeA, nodeB) {
    const parentA = nodeA.parentNode;
    const siblingA = nodeA.nextSibling === nodeB ? nodeA : nodeA.nextSibling;

    // Move `nodeA` to before the `nodeB`
    nodeB.parentNode.insertBefore(nodeA, nodeB);

    // Move `nodeB` to before the sibling of `nodeA`
    parentA.insertBefore(nodeB, siblingA);
};

function resizeCanvas(canvas, pad) {
    const ratio = Math.max(window.devicePixelRatio || 1, 1);
    canvas.width = canvas.offsetWidth * ratio;
    canvas.height = canvas.offsetHeight * ratio;
    canvas.getContext("2d").scale(ratio, ratio);
    pad.clear(); // otherwise isEmpty() might return incorrect value
}

async function URLtoBase64(url) {
    return new Promise((resolve, reject) => {
        fetch(url)
            .then(response => response.blob())
            .then(blob => {
                const reader = new FileReader();
                reader.readAsDataURL(blob);
                reader.onloadend = () => {
                    const base64data = reader.result;
                    resolve(base64data);
                };
            });
    });
}

window.addEventListener('DOMContentLoaded', async function () {

    history.replaceState("home", "", document.location.href);

    if (pseudo != "") {
        updateOnLogin("sign");
    }

    updateTemplateOnLogin();

    document.getElementById("template_worker").addEventListener("change", function () {
        if (document.getElementById("template_worker").value == "emptyAll") {
            document.getElementById("input_firstName_worker").value = "";
            document.getElementById("input_lastName_worker").value = "";
            document.getElementById("input_siren_worker").value = "";
            document.getElementById("input_address_worker").textContent = "";
        } else {
            var templateObj = user_templates.workers.find(function (worker) {
                return worker.siren === document.getElementById("template_worker").value;
            });
            document.getElementById("input_firstName_worker").value = templateObj.firstName;
            document.getElementById("input_lastName_worker").value = templateObj.lastName;
            document.getElementById("input_siren_worker").value = templateObj.siren;
            document.getElementById("input_address_worker").textContent = templateObj.address;
        }
    });

    document.getElementById("template_customer").addEventListener("change", function () {
        if (document.getElementById("template_customer").value == "emptyAll") {
            document.getElementById("input_name_customer").value = "";
            document.getElementById("input_siren_customer").value = "";
            document.getElementById("input_address_customer").textContent = "";
        } else {
            var templateObj = user_templates.customers.find(function (customer) {
                return customer.siren === document.getElementById("template_customer").value;
            });
            document.getElementById("input_name_customer").value = templateObj.name;
            document.getElementById("input_siren_customer").value = templateObj.siren;
            document.getElementById("input_address_customer").textContent = templateObj.address;
        }


    });

    document.addEventListener("click", async (event) => {
        if (event.target.getAttribute("route") || event.target.parentElement.getAttribute("route")) {
            // Prevent a new page from loading
            event.preventDefault();
            var route;
            if (event.target.getAttribute("route")) {
                route = event.target.getAttribute("route");
            } else {
                route = event.target.parentElement.getAttribute("route");
            }
            updateView(route);
            if (route == "home") {
                route = "/";
            }
            if (rootRoute != null && route != "/") {
                route = rootRoute + "/" + route;
            } else if (rootRoute != null) {
                route = "/" + rootRoute;
            }
            history.pushState(route, "", route);
        }
    });

    // Handle forward/back buttons
    window.addEventListener("popstate", (event) => {
        console.log(event);
        // If a state has been provided, we have a "simulated" page
        // and we update the current page.
        if (event.state) {
            // Simulate the loading of the previous page
            updateView(event.state.replace(rootRoute + "/", ""));
        }
    });


    document.addEventListener("mousemove", (e) => {
        rotateElement(e, document.getElementById("img_paralax_1_1"), 2, 50);
        rotateElement(e, document.getElementById("img_paralax_1_2"), 1, 310);
        rotateElement(e, document.getElementById("img_paralax_1_3"), 3, 70);
    });

    document.querySelectorAll('.landing_workflow_nav_element').forEach(element => {
        element.addEventListener('click', () => { landingWorkflowContentSwitcher(element) });
    });

    document.getElementById("switch_toggle_light").addEventListener("click", () => {
        toggleLightMode(document.getElementById("switch_toggle_light"));
    }, false);

    document.getElementById("switch_toggle_light_min").addEventListener("click", () => {
        toggleLightMode(document.getElementById("switch_toggle_light_min"));
    }, false);

    document.getElementById("nav_menu_button").addEventListener("click", () => {
        document.getElementById("absolute_min_menu").style.right = "0%";
    });

    document.getElementById("exit_menu").addEventListener("click", () => {
        document.getElementById("absolute_min_menu").style.right = "-100%";
    });

    document.querySelectorAll('.input_input').forEach(element => {
        element.addEventListener('input', () => { inputError(element) });
        element.addEventListener('focus', () => { inputError(element) });
        element.addEventListener('focusout', () => { showHideError(null, element) });
    });

    document.querySelectorAll('.input_phone_number').forEach(element => {
        element.addEventListener('input', () => { inputError(element) });
        element.addEventListener('focus', () => { inputError(element) });
        element.addEventListener('focusout', () => { showHideError(null, element.parentElement) });
    });

    document.querySelectorAll('.devis_add_button_icon').forEach(element => {
        element.addEventListener('click', () => { addDevisLine() });
    });

    document.querySelectorAll('.invert_toggle_input').forEach(element => {
        element.addEventListener('click', () => { toglleCheck(element) });
    });

    document.getElementById("button_clear_signature").addEventListener('click', () => { signaturePad1.clear(); });

    document.getElementById("button_generatePDF").addEventListener('click', () => { generatePDF() });

    document.getElementById("button_delete_worker").addEventListener('click', () => { deleteTemplate("worker") });

    document.getElementById("button_delete_customer").addEventListener('click', () => { deleteTemplate("customer") });

    document.getElementById("button_login").addEventListener('click', () => { login_start() });

    document.getElementById("button_unsign").addEventListener('click', () => { unsign() });

    document.getElementById("button_register").addEventListener('click', () => { register_start() });

    document.getElementById("button_save_worker").addEventListener('click', () => { saveTemplate("worker") });

    document.getElementById("button_save_customer").addEventListener('click', () => { saveTemplate("customer") });

    document.querySelectorAll('.devis_price_calc').forEach(element => {
        element.addEventListener('input', () => { priceCalc(element) });
    });


    document.querySelectorAll('.svg_devis_line_dots').forEach(dots => {
        dots.addEventListener('mousedown', DragOn);
    });

    document.querySelectorAll('.devis_line_buttons_container').forEach(trash => {
        trash.addEventListener('dragenter', (event) => {
            event.target.style.background = "red";
        });
    });

    articleCreateNavButtons();

    document.querySelectorAll('.readArticle').forEach(element => {
        element.addEventListener('click', () => { revealArticle(element) });
    });

    var datePlus30 = new Date();
    datePlus30.setDate(datePlus30.getDate() + 30);
    document.getElementById('date_emission').valueAsDate = new Date();
    document.getElementById('date_paiement').valueAsDate = datePlus30;

    updateView(originRoute, true);
});

window.addEventListener('load', function () {
    signaturePad1 = new SignaturePad(document.querySelector("#canvas1"));
    resizeCanvas(document.querySelector("#canvas1"), signaturePad1);
    /*window.addEventListener("resize", function (e) {
        resizeCanvas(document.querySelector("#canvas1"), signaturePad1);
    });*/
    document.getElementById("content_container").style.display = "none";
});

window.addEventListener('mousemove', (event) => {
    cursorX = event.pageX;
    cursorY = event.pageY;
});

function rotateElement(event, element, tempSensi, originPositionX) {
    //sensibility
    const sensibility = 30;
    // get mouse position
    const x = event.clientX - 250;
    const y = event.clientY;
    // console.log(x, y)

    // find the middle
    const middleX = window.innerWidth / 2;
    const middleY = window.innerHeight / 2;
    // console.log(middleX, middleY)

    // get offset from middle as a percentage
    // and tone it down a little
    const offsetX = ((x - middleX) / middleX) * sensibility;
    const offsetY = ((y - middleY) / middleY) * sensibility;
    // console.log(offsetX, offsetY);

    // set rotation
    element.style.setProperty("--rotateX", offsetX + "deg");
    element.style.setProperty("--rotateY", -1 * offsetY + "deg");

    //function by Kevin Powell -> https://www.youtube.com/watch?v=Z-3tPXf9a7M

    element.style.left = originPositionX + ((offsetX * tempSensi)) + "px";
}

function getCssProperty(element, property) {
    return window.getComputedStyle(element, null).getPropertyValue(property);
}

function addDevisLine() {
    var firstLine_node = document.getElementById("first_devis_lines_container").firstElementChild;
    var cloneLine_node = firstLine_node.cloneNode(true);
    document.getElementById('first_devis_lines_container').appendChild(cloneLine_node);
    cloneLine_node.querySelector(".devis_description").innerHTML = "";
    cloneLine_node.querySelector(".devis_quantity").value = "";
    cloneLine_node.querySelector(".devis_price").value = "";
    cloneLine_node.querySelector(".devis_price_total").innerHTML = "0€";
    cloneLine_node.removeAttribute('id');
    cloneLine_node.querySelector(".svg_devis_line_dots").addEventListener('mousedown', DragOn);
    cloneLine_node.querySelectorAll('.devis_price_calc').forEach(element => {
        element.addEventListener('input', () => { priceCalc(element) });
    });
}

function onElement(element) {
    var ePosition = element.getBoundingClientRect();
    if (ePosition.x < cursorX && cursorX < ePosition.right && (ePosition.y + window.scrollY) < cursorY && cursorY < (ePosition.bottom + window.scrollY)) {
        return true;
    } else {
        return false;
    }
}

function priceCalc(element) {
    var parent = element.parentElement;
    var quantity = parent.querySelector('.devis_quantity').value;
    var price = parent.querySelector('.devis_price').value;
    var totalLine = 0;
    if (quantity > 0 && price > 0 && quantity != "" && price != "") {
        totalLine = quantity * price;
    }
    parent.querySelector('.devis_price_total').innerHTML = totalLine.toFixed(2) + "€"

    var totalDevis = 0;
    parent.parentElement.childNodes.forEach((line) => {
        if (line.className == 'devis_line') {
            totalDevis = (totalDevis + parseFloat(line.querySelector(".devis_price_total").innerHTML.replace("€", "")));
        }
    })
    document.getElementById("price_indicator_htc").innerHTML = totalDevis + "€";
    document.getElementById("price_indicator_ttc").innerHTML = (parseFloat(totalDevis) + (parseFloat(totalDevis) * 0.2)).toFixed(2) + "€";
    if (totalDevis <= 150) {
        document.getElementById("toggle_tva").style.display = "none";
    } else {
        document.getElementById("toggle_tva").style.display = "flex";
    }
}

function toglleCheck(element) {
    if (element.id == "switch_tva") {
        if (element.checked) {
            document.getElementById('price_indicator_ttc').style.textDecoration = "none";
            document.getElementById('tva_container').style.opacity = "1";

        } else {
            document.getElementById('price_indicator_ttc').style.textDecoration = "line-through";
            document.getElementById('tva_container').style.opacity = "0.5";
        }
    } else if (element.id == "switch_artisan") {
        if (element.checked) {
            document.getElementById('input_container_artisan_rm_worker').style.height = "auto";
            document.getElementById('input_container_artisan_rm_worker').style.marginTop = "1rem";
            document.getElementById('input_container_artisan_rm_worker').style.overflow = "visible";
        } else {
            document.getElementById('input_container_artisan_rm_worker').style.height = "0";
            document.getElementById('input_container_artisan_rm_worker').style.marginTop = "0";
            document.getElementById('input_container_artisan_rm_worker').style.overflow = "hidden";
        }

    } else if (element.id == "switch_commerce") {
        if (element.checked) {
            document.getElementById('input_container_commercant_rcs_worker').style.height = "auto";
            document.getElementById('input_container_commercant_rcs_worker').style.marginTop = "1rem";
            document.getElementById('input_container_commercant_rcs_worker').style.overflow = "visible";
        } else {
            document.getElementById('input_container_commercant_rcs_worker').style.height = "0";
            document.getElementById('input_container_commercant_rcs_worker').style.marginTop = "0";
            document.getElementById('input_container_commercant_rcs_worker').style.overflow = "hidden";
        }
    }

}

function generatePDF() {
    alertDisplay("waiting", "Génération en cours...");
    // Modify all PDF values with inputs

    //global infos
    document.getElementById("header_to_generate_subtitle").innerHTML = document.getElementById("input_devis_title").value.trim();
    var inputDate = new Date(document.getElementById("date_emission").value);
    var inputDateFrenchFormat = inputDate.toLocaleDateString('en-GB');
    document.getElementById("header_to_generate_date_value").innerHTML = inputDateFrenchFormat;
    var billNumber = (document.getElementById("input_devis_number").value.toString().length == 1 ? "00" : document.getElementById("input_devis_number").value.toString().length == 2 ? "0" : "") + document.getElementById("input_devis_number").value.toString();
    var month = (parseInt(inputDate.getMonth()) + 1).toString();
    document.getElementById("header_to_generate_number").innerHTML = "N°" + inputDate.getFullYear() + (month.length == 1 ? "0" : "") + month + (inputDate.getDate().toString().length == 1 ? "0" : "") + inputDate.getDate() + billNumber;

    //worker infos
    document.getElementById("section_to_generate_id_worker_name").innerHTML = document.getElementById("input_firstName_worker").value + " " + document.getElementById("input_lastName_worker").value;
    document.getElementById("section_to_generate_id_worker_address").innerHTML = document.getElementById("input_address_worker").innerHTML;
    document.getElementById("section_to_generate_id_worker_siren").innerHTML = document.getElementById("input_siren_worker").value;
    if (document.getElementById("toggle_commercant_label").checked) {
        document.getElementById("section_to_generate_id_worker_rcs").style.display = "block";
        document.getElementById("section_to_generate_id_worker_rcs_value") = document.getElementById("input_rcs").value;
    }
    if (document.getElementById("toggle_artisan_label").checked) {
        document.getElementById("section_to_generate_id_worker_rm").style.display = "block";
        document.getElementById("section_to_generate_id_worker_rm_value") = document.getElementById("input_rm").value;
    }

    //customer infos
    document.getElementById("section_to_generate_id_customer_name").innerHTML = document.getElementById("input_name_customer").value;
    document.getElementById("section_to_generate_id_customer_address").innerHTML = document.getElementById("input_address_customer").innerHTML;
    document.getElementById("section_to_generate_id_customer_siren").innerHTML = document.getElementById("input_siren_customer").value;

    //devis elements
    var totalDevis = 0;
    var firstLine_node = document.getElementById("section_devis_to_generate_line_1");
    var cloneLine_node;
    document.getElementById("first_devis_lines_container").childNodes.forEach((line) => {
        if (line.className == 'devis_line' && line.querySelector(".devis_description").textContent != "" && line.querySelector(".devis_price").value != "" && line.querySelector(".devis_quantity").value != "") {
            totalDevis = totalDevis + parseFloat(line.querySelector(".devis_price").value) * parseFloat(line.querySelector(".devis_quantity").value);
            cloneLine_node = firstLine_node.cloneNode(true);
            document.getElementById("section_devis_to_generate").appendChild(cloneLine_node);
            cloneLine_node.removeAttribute('id');
            cloneLine_node.style.display = "flex";

            cloneLine_node.querySelector(".section_devis_to_generate_line_description").innerHTML = line.querySelector(".devis_description").innerHTML;
            cloneLine_node.querySelector(".section_devis_to_generate_line_quantity").innerHTML = line.querySelector(".devis_quantity").value;
            cloneLine_node.querySelector(".section_devis_to_generate_line_unit_price").innerHTML = line.querySelector(".devis_price").value + "€";
            cloneLine_node.querySelector(".section_devis_to_generate_line_total_price").innerHTML = parseFloat(line.querySelector(".devis_price").value) * parseFloat(line.querySelector(".devis_quantity").value) + "€";
        }
        document.getElementById("section_devis_to_generate_line_1").style.display = "none";
        document.getElementById("section_to_generate_id_total_htc").innerHTML = totalDevis + "€";
        if (document.getElementById("switch_tva").checked && totalDevis > 150) {
            document.getElementById("section_to_generate_id_total_ttc").innerHTML = (totalDevis + (totalDevis * 0.2)) + "€";
            document.getElementById("section_devis_to_generate_tva_exempt_text").style.display = "none";
        } else {
            document.getElementById("section_to_generate_id_tva").innerHTML = "0%";
            document.getElementById("section_to_generate_id_total_ttc").innerHTML = totalDevis + "€";
            document.getElementById("section_devis_to_generate_tva_exempt_text").style.display = "block";
        }
    });

    //date infos
    document.getElementById("section_to_generate_id_taux_interet").innerHTML = document.getElementById("input_indemnite").value;
    inputDate = new Date(document.getElementById("date_paiement").value);
    inputDate = inputDate.toLocaleDateString('en-GB');
    document.getElementById("section_to_generate_id_date_limit").innerHTML = inputDate;

    //payements infos
    if (document.getElementById("input_iban").value.trim() != "" || document.getElementById("input_payement_link").value.trim() != "") {
        document.getElementById("section_to_generate_info_paiement").style.display = "flex";
    }
    if (document.getElementById("input_iban").value.trim() != "") {
        document.getElementById("section_to_generate_info_paiement_iban_container").style.display = "flex";
        document.getElementById("section_to_generate_label_iban_name").innerHTML = document.getElementById("input_firstName_worker").value + " " + document.getElementById("input_lastName_worker").value;
        document.getElementById("section_to_generate_label_iban").innerHTML = document.getElementById("input_iban").value;
    }
    if (document.getElementById("input_payement_link").value.trim() != "") {
        document.getElementById("section_to_generate_internet_paiement_button").style.display = "block";
    }

    //pdf generation
    var element = document.getElementById("absolute_to_generate");
    var signature = document.getElementById('canvas1');
    var signature_width = signature.width / (signature.height / 50);
    var pdf = new jsPDF('p', 'pt', 'a4');
    //width 600px * 849px for A4 page
    document.getElementById("absolute_to_generate").style.display = "block";
    pdf.html(element)
        .then(() => {
            pdf.addImage(signature, 'PNG', 60, document.getElementById('section_to_generate_id_retard').offsetTop + document.getElementById('section_to_generate_id_retard').offsetHeight + 40, signature_width, 80, "signature", "NONE", 10);
            pdf.setFontSize(12);
            pdf.setTextColor(255, 255, 255);
            pdf.setFont('Helvetica', 'bold')
            if (document.getElementById("input_payement_link").value.trim() != "") {
                pdf.textWithLink('Payer en ligne', document.getElementById('section_to_generate_internet_paiement_button').offsetLeft + 10, getElementOffset(document.getElementById('section_to_generate_internet_paiement_button')).top + 16.5, { url: document.getElementById("input_payement_link").value.trim() });
            }
            pdf.save('fileName.pdf');
            document.getElementById("absolute_to_generate").style.display = "none";
            alertDisplay("success", "PDF généré avec succès !");
        }
        );
}

function login_start() {
    if (document.getElementById("input_login").value.trim() != "" && document.getElementById("input_password").value.trim() != "") {
        alertDisplay("waiting", "Connexion en cours...");
        var fetchBody = {
            "type": "login"
        }
        fetchCommon("fetch", fetchBody);
    }
}

function unsign() {
    alertDisplay("waiting", "Déconnexion en cours...");
    var fetchBody = {
        "type": "unsign"
    }
    fetchCommon("fetch", fetchBody);
}

function login_challenge(salt) {
    if (document.getElementById("input_login").value.trim() != "" && document.getElementById("input_password").value.trim() != "") {

        sha256(document.getElementById("input_password").value.trim()).then(hash => {
            console.log('Hash SHA-256 de "test":', hash);
            sha256(hash + salt).then(hash2 => {
                console.log('Hash SHA-256 de hash test + hash salt:', hash2);
                var fetchBody = {
                    "type": "login",
                    "login": document.getElementById("input_login").value.trim(),
                    "challenge": hash2
                }
                fetchCommon("fetch", fetchBody);
            }).catch(error => {
                console.error('Erreur lors du calcul du hachage:', error);
            });
        }).catch(error => {
            console.error('Erreur lors du calcul du hachage:', error);
        });

    }
}

function register_start() {
    if (document.getElementById("input_login").value.trim() != "" && document.getElementById("input_password").value.trim() != "") {
        sha256(document.getElementById("input_password").value.trim()).then(hash => {
            var fetchBody = {
                "type": "register",
                "login": document.getElementById("input_login").value.trim(),
                "password": hash
            }
            fetchCommon("fetch", fetchBody);
        }).catch(error => {
            console.error('Erreur lors du calcul du hachage:', error);
        });
    }
}

function saveTemplate(type) {
    var match = false;
    var typeObj = {};
    if (type == "worker") {
        typeObj = user_templates.workers;
    }
    if (type == "customer") {
        typeObj = user_templates.customers;
    }
    typeObj.forEach(function (obj) {
        if (obj.siren == document.getElementById("input_siren_" + type).value.trim()) {
            match = true;
            if (type == "customer") {
                obj.name = document.getElementById("input_name_" + type).value.trim();
                obj.siren = document.getElementById("input_siren_" + type).value.trim();
                obj.address = document.getElementById("input_address_" + type).textContent.trim();
                obj.phone = document.getElementById("input_phone_prefix_" + type).value + document.getElementById("input_phone_number_" + type).value.trim();
            } else {
                obj.firstName = document.getElementById("input_firstName_" + type).value.trim();
                obj.lastName = document.getElementById("input_lastName_" + type).value.trim();
                obj.siren = document.getElementById("input_siren_" + type).value.trim();
                obj.address = document.getElementById("input_address_" + type).textContent.trim();
                obj.phone = document.getElementById("input_phone_prefix_" + type).value + document.getElementById("input_phone_number_" + type).value.trim();
            }
            obj.address = document.getElementById("input_address_" + type).textContent.trim();
            obj.phone = document.getElementById("input_phone_prefix_" + type).value + document.getElementById("input_phone_number_" + type).value.trim();
        }
    });
    if (!match) {
        if (type == "customer") {
            var tempObj = {
                "name": document.getElementById("input_name_" + type).value.trim(),
                "siren": document.getElementById("input_siren_" + type).value.trim(),
                "address": document.getElementById("input_address_" + type).textContent.trim(),
                "phone": document.getElementById("input_phone_prefix_" + type).value + document.getElementById("input_phone_number_" + type).value.trim(),
            };
            // Création d'un nouvel élément <option>
            var nouvelOption = document.createElement("option");
            // Attribution de la valeur et du texte à l'option
            nouvelOption.value = tempObj.siren;
            nouvelOption.text = tempObj.name;
            document.getElementById("template_worker").add(nouvelOption);
        } else {
            var tempObj = {
                "firstName": document.getElementById("input_firstName_" + type).value.trim(),
                "lastName": document.getElementById("input_lastName_" + type).value.trim(),
                "siren": document.getElementById("input_siren_" + type).value.trim(),
                "address": document.getElementById("input_address_" + type).textContent.trim(),
                "phone": document.getElementById("input_phone_prefix_" + type).value + document.getElementById("input_phone_number_" + type).value.trim(),
            };
            // Création d'un nouvel élément <option>
            var nouvelOption = document.createElement("option");
            // Attribution de la valeur et du texte à l'option
            nouvelOption.value = tempObj.siren;
            nouvelOption.text = tempObj.firstName + " " + tempObj.lastName;
            document.getElementById("template_worker").add(nouvelOption);
        }
        typeObj.push(tempObj);
    }
    console.log(user_templates);
    //localStorage.setItem("user_templates", JSON.stringify(user_templates));
    var fetchBody = {
        "type": "templateSave",
        "templates": JSON.stringify(user_templates)
    }
    fetchCommon("fetch", fetchBody);
    alertDisplay("waiting", "Sauvegarde en cours...");
}

function deleteTemplate(type) {
    if (type == "worker") {
        const indexWorkerToDelete = user_templates.workers.findIndex(worker => worker.siren === document.getElementById("input_siren_" + type).value.trim());
        // Si un élément est trouvé, le supprimer du tableau
        if (indexWorkerToDelete !== -1) {
            user_templates.workers.splice(indexWorkerToDelete, 1);
        }
    }
    if (type == "customer") {
        const indexCustomerToDelete = user_templates.customers.findIndex(customer => customer.siren === document.getElementById("input_siren_" + type).value.trim());
        // Si un élément est trouvé, le supprimer du tableau
        if (indexCustomerToDelete !== -1) {
            user_templates.customers.splice(indexCustomerToDelete, 1);
        }
    }

    var fetchBody = {
        "type": "templateSave",
        "templates": JSON.stringify(user_templates)
    }
    fetchCommon("fetch", fetchBody);
    supprimerOptionParValeur('template_' + type, document.getElementById("input_siren_" + type).value.trim());
    forcerOptionSelect('template_' + type, 'emptyAll');
    alertDisplay("waiting", "Sauvegarde en cours...");
}

async function sha256(message) {
    // Convertir la chaîne de caractères en un tableau d'octets (ArrayBuffer)
    const buffer = new TextEncoder().encode(message);

    // Calculer le hachage SHA-256
    const hashBuffer = await crypto.subtle.digest('SHA-256', buffer);

    // Convertir le tableau d'octets en une chaîne hexadécimale
    const hashArray = Array.from(new Uint8Array(hashBuffer));
    const hashHex = hashArray.map(byte => byte.toString(16).padStart(2, '0')).join('');

    return hashHex;
}

function getElementOffset(element) {
    const boundingRect = element.getBoundingClientRect();
    return {
        left: boundingRect.left + window.scrollX,
        top: boundingRect.top + window.scrollY
    };
}

function updateView(viewName, except = false) {
    if (window.location.pathname.replace("/", "") != viewName || except) {
        document.getElementById("loading_container").classList.remove("wraped");
        setTimeout(() => {
            document.querySelectorAll('.full_page_container').forEach(element => {
                element.style.display = "none";
            });
            document.getElementById("loading_container").classList.add("wraped");

            if (viewName == "home" || viewName == "/") {
                document.getElementById("landing_page").style.display = "flex";
            } else if (viewName == "generator") {
                document.getElementById("content_container").style.display = "flex";
                setTimeout(() => {
                    resizeCanvas(document.querySelector("#canvas1"), signaturePad1);
                }, 1000);
            } else if (viewName == "story") {
                document.getElementById("story").style.display = "flex";
            } else if (viewName == "roadmap") {
                document.getElementById("roadmap").style.display = "flex";
            }
            document.getElementById("loading_container").classList.add("wraped");
        }, 1500);
    }


}

function landingWorkflowContentSwitcher(button) {
    var numberID = button.id.replace("landing_workflow_nav_element_", "");
    document.querySelector(".landing_workflow_nav_element_active").classList.remove("landing_workflow_nav_element_active");
    button.classList.add("landing_workflow_nav_element_active");
    document.querySelector(".landing_workflow_content_container_active").classList.add("landing_workflow_content_container");
    document.querySelector(".landing_workflow_content_container_active").classList.remove("landing_workflow_content_container_active");
    document.getElementById("landing_workflow_content_container_" + numberID).classList.add("landing_workflow_content_container_active");
    document.getElementById("landing_workflow_content_container_" + numberID).classList.remove("landing_workflow_content_container");
}

function fetchCommon(uri, body) {
    // Remplacez 'https://api.example.com/user' par l'URL de l'API que vous souhaitez interroger
    const requestOptions = {
        method: 'POST', // Méthode de la requête
        headers: {
            'Content-Type': 'application/json' // En-têtes de la requête
        },
        body: JSON.stringify(body)
    };

    fetch(window.location.origin + "/" + uri, requestOptions)
        .then(response => {
            // Vérifie si la réponse est OK (code 200)
            if (!response.ok) {
                throw new Error('Erreur réseau : ' + response.status);
                alertDisplay("error", "Erreur après : " + response.status);
            }
            // Parse la réponse JSON
            return response.json();
        })
        .then(data => {
            // Faites quelque chose avec les données récupérées
            console.log(data);
            if (data.type == "challengeAccepted") {
                login_challenge(data.salt);
            }
            if (data.type == "login" && data.status == "success") {
                //location.reload();
                pseudo = data.pseudo;
                user_templates = data.templates;
                updateTemplateOnLogin();
                updateOnLogin("sign");
            } else if (data.type == "unsign" && data.status == "success") {
                pseudo = "";
                user_templates = "";
                updateOnLogin("unsign");
            } else if (data.type == "register" && data.status == "success") {
                location.reload();
            } else if (data.type == "templateSave" && data.status == "success") {
                alertDisplay("success", "Template sauvegardé");
            }
        })
        .catch(error => {
            // Attrape les erreurs possibles
            console.error('Erreur lors de la récupération des données :', error);
        });
}

function alertDisplay(type, message) {
    document.getElementById("alertDisplayText").innerHTML = message;
    if (type == "success") {
        document.getElementById("alertDisplay").style.backgroundColor = "var(--valid-color)";
    } else if (type == "error") {
        document.getElementById("alertDisplay").style.backgroundColor = "var(--error-color)";
    } else if (type == "waiting") {
        document.getElementById("alertDisplay").style.backgroundColor = "var(--inactive-second-color)";
    }
    document.getElementById("alertDisplay").style.height = "30px";

    if (type != "waiting") {
        setTimeout(() => {
            document.getElementById("alertDisplay").style.height = "0px";
        }, 3000);
    }
}

function forcerOptionSelect(elementId, valeur) {
    var selectElement = document.getElementById(elementId);
    selectElement.value = valeur;

    // Optionnel : déclencher l'événement 'change' si nécessaire
    var event = new Event('change');
    selectElement.dispatchEvent(event);
}

function supprimerOptionParValeur(selectId, valeur) {
    var selectElement = document.getElementById(selectId);
    var options = selectElement.options;

    for (var i = 0; i < options.length; i++) {
        if (options[i].value === valeur) {
            selectElement.removeChild(options[i]);
            break;
        }
    }
}

function updateOnLogin(type) {
    if (type == "sign") {
        document.querySelectorAll('.withoutLoginElement').forEach(element => {
            element.style.display = "none";
        });
        document.querySelectorAll('.withLoginElement').forEach(element => {
            element.style.display = "flex";
        });
        document.getElementById("display_pseudo").innerHTML = pseudo;
        alertDisplay("success", "Connecté");
        document.getElementById("input_login").value = "";
        document.getElementById("input_password").value = "";
    } else if (type == "unsign") {
        document.querySelectorAll('.withoutLoginElement').forEach(element => {
            element.style.display = "flex";
        });
        document.querySelectorAll('.withLoginElement').forEach(element => {
            element.style.display = "none";
        });
        document.getElementById('template_worker').innerHTML = '';
        document.getElementById('template_customer').innerHTML = '';
        var nouvelOption = document.createElement("option");
        // Attribution de la valeur et du texte à l'option
        nouvelOption.value = "emptyAll";
        nouvelOption.text = "Template vide";
        nouvelOption2 = nouvelOption.cloneNode(true);
        document.getElementById("template_customer").add(nouvelOption);
        document.getElementById("template_worker").add(nouvelOption2);
        alertDisplay("success", "Déconnecté");
    }

}

function updateTemplateOnLogin() {
    if (user_templates != "") {
        user_templates = JSON.parse(user_templates);
        user_templates.workers.forEach(function (obj) {
            // Création d'un nouvel élément <option>
            var nouvelOption = document.createElement("option");
            // Attribution de la valeur et du texte à l'option
            nouvelOption.value = obj.siren;
            nouvelOption.text = obj.firstName + " " + obj.lastName;
            document.getElementById("template_worker").add(nouvelOption);
        });

        user_templates.customers.forEach(function (obj) {
            // Création d'un nouvel élément <option>
            var nouvelOption = document.createElement("option");
            // Attribution de la valeur et du texte à l'option
            nouvelOption.value = obj.siren;
            nouvelOption.text = obj.name;
            document.getElementById("template_customer").add(nouvelOption);
        });
    } else {
        user_templates = {
            customers: [],
            workers: [],
            devis: []
        }
    }

}

function articleCreateNavButtons() {
    var button = document.createElement('div');
    button.classList.add('button_basic');
    button.classList.add('button_basic_invert');
    document.querySelectorAll('.article_chapter_title').forEach(element => {
        var cloneButton = button.cloneNode(true);
        cloneButton.innerHTML = element.innerHTML;
        element.parentElement.parentElement.parentElement.querySelector('.article_nav_container').querySelector('.article_nav').appendChild(cloneButton);
        cloneButton.addEventListener('click', function () {
            element.scrollIntoView({ behavior: 'smooth' });
        })
    });
}

function revealArticle(element) {
    var article = element.parentElement.parentElement.parentElement.parentElement.querySelector(".article_container");
    if (article.style.height == "auto") {
        article.style.height = "0px";
        article.style.marginTop = "0px";
        article.style.overflow = "hidden";
    } else {
        console.log("test else");
        article.style.height = "auto";
        article.style.marginTop = "25px";
        article.style.overflow = "clip";
    }

}