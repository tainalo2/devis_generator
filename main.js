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
            if (route == "home"){
                route = "/";
            }
            console.log("route 1 :" + route);
            if (rootRoute != null && route != "/" ) {
                console.log("route 2 :" + route);
                route = rootRoute + "/" + route;
            } else if (rootRoute != null) {
                console.log("route 3 :" + route);
                console.log("rootRoute : " + rootRoute);
                route = rootRoute;
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

    var datePlus30 = new Date();
    datePlus30.setDate(datePlus30.getDate() + 30);
    document.getElementById('date_emission').valueAsDate = new Date();
    document.getElementById('date_paiement').valueAsDate = datePlus30;




    //Check localStorage + set default
    localStorage.clear();
    if (localStorage.getItem("image_animate_writing") === null) {
        localStorage.setItem("image_animate_writing", await URLtoBase64('https://raw.githubusercontent.com/tainalo2/devis_generator/main/src/image/animate_writing.png'));
    }
    if (localStorage.getItem("image_animate_angry") === null) {
        localStorage.setItem("image_animate_angry", await URLtoBase64('https://raw.githubusercontent.com/tainalo2/devis_generator/main/src/image/animate_angry.png'));
    }
    if (localStorage.getItem("image_landing_paralax_1_1") === null) {
        localStorage.setItem("image_landing_paralax_1_1", await URLtoBase64('https://raw.githubusercontent.com/tainalo2/devis_generator/main/src/image/landing_paralax_1_1.png'));
    }
    if (localStorage.getItem("image_landing_paralax_1_2") === null) {
        localStorage.setItem("image_landing_paralax_1_2", await URLtoBase64('https://raw.githubusercontent.com/tainalo2/devis_generator/main/src/image/landing_paralax_1_2.png'));
    }
    if (localStorage.getItem("image_landing_paralax_1_3") === null) {
        localStorage.setItem("image_landing_paralax_1_3", await URLtoBase64('https://raw.githubusercontent.com/tainalo2/devis_generator/main/src/image/landing_paralax_1_3.png'));
    }

    document.querySelectorAll('.error_img').forEach(errorIMG => {
        errorIMG.src = localStorage.getItem("image_animate_angry");
    });

    document.getElementById("main_title_img").src = localStorage.getItem("image_animate_writing");
    document.getElementById("img_paralax_1_1").src = localStorage.getItem("image_landing_paralax_1_1");
    document.getElementById("img_paralax_1_2").src = localStorage.getItem("image_landing_paralax_1_2");
    document.getElementById("img_paralax_1_3").src = localStorage.getItem("image_landing_paralax_1_3");

    updateView(originRoute);

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
            totalDevis = (totalDevis + parseFloat(line.querySelector(".devis_price_total").innerHTML.replace("€", ""))).toFixed(2);
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
    document.getElementById("section_to_generate_id_worker_address").innerHTML = document.getElementById("input_address_worker").textContent.trim();
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
    document.getElementById("section_to_generate_id_customer_address").innerHTML = document.getElementById("input_address_customer").textContent.trim();
    document.getElementById("section_to_generate_id_customer_siren").innerHTML = document.getElementById("input_siren_customer").value;

    //devis elements
    var totalDevis = 0;
    var firstLine_node = document.getElementById("section_devis_to_generate_line_1");
    var cloneLine_node;
    document.getElementById("first_devis_lines_container").childNodes.forEach((line) => {
        if (line.className == 'devis_line' && line.querySelector(".devis_description").textContent.trim() != "" && line.querySelector(".devis_price").value != "" && line.querySelector(".devis_quantity").value != "") {
            totalDevis = totalDevis + parseFloat(line.querySelector(".devis_price").value) * parseFloat(line.querySelector(".devis_quantity").value);
            cloneLine_node = firstLine_node.cloneNode(true);
            document.getElementById("section_devis_to_generate").appendChild(cloneLine_node);
            cloneLine_node.removeAttribute('id');
            cloneLine_node.style.display = "flex";

            cloneLine_node.querySelector(".section_devis_to_generate_line_description").innerHTML = line.querySelector(".devis_description").textContent.trim();
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
        }
        );
}

function getElementOffset(element) {
    const boundingRect = element.getBoundingClientRect();
    return {
        left: boundingRect.left + window.scrollX,
        top: boundingRect.top + window.scrollY
    };
}

function updateView(viewName) {
    if (viewName == "home" || viewName == "/") {
        document.getElementById("loading_container").classList.remove("wraped");
        setTimeout(() => {
            document.getElementById("content_container").style.display = "none";
            document.getElementById("landing_page").style.display = "flex";
            document.getElementById("loading_container").classList.add("wraped");
        }, 1500);

    } else if (viewName == "generator") {
        document.getElementById("loading_container").classList.remove("wraped");
        setTimeout(() => {
            document.getElementById("landing_page").style.display = "none";
            document.getElementById("content_container").style.display = "flex";
            setTimeout(() => {
                resizeCanvas(document.querySelector("#canvas1"), signaturePad1);
            }, 1000);
            
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