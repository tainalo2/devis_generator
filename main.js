window.jsPDF = window.jspdf.jsPDF
window.html2canvas = html2canvas;

const regexFirstLastName = new RegExp("([A-ZÀ-ÿ][-,a-z. ']+[ ]*)+");
const regexSiren = new RegExp("^\\d{9}$");
const regexEmail = new RegExp("^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$")
const regexNumber = new RegExp("[0-9]");
const regexIntFloat = new RegExp("[0-9]*\.?[0-9]*")
const regexTwoNumbers = new RegExp("[0-9]{3}");
const regexPhoneNumber = new RegExp("^(([0-9]{2}-){4})([0-9]{2})$");
let signaturePad1;

function toggleLightMode(element) {
    if (element.checked) {
        document.querySelector(':root').style.setProperty('--main-bg-color', 'black');
        document.querySelector(':root').style.setProperty('--main-second-color', 'white');
        document.querySelector(':root').style.setProperty('--main-tier_color', 'rgb(53, 53, 53)');
        document.querySelector(':root').style.setProperty('--main-quater_color', 'rgb(248, 248, 248)');
        document.querySelector(':root').style.setProperty('--filter-main-color-svg', 'invert(100%) sepia(2%) saturate(906%) hue-rotate(227deg) brightness(121%) contrast(100%)');
        document.querySelector(':root').style.setProperty('--filter-second-color-svg', 'invert(0%) sepia(2%) saturate(906%) hue-rotate(227deg) brightness(121%) contrast(100%)');
        signaturePad1.penColor = "white";
    } else {
        document.querySelector(':root').style.setProperty('--main-bg-color', 'white');
        document.querySelector(':root').style.setProperty('--main-second-color', 'black');
        document.querySelector(':root').style.setProperty('--main-tier_color', 'rgb(248, 248, 248)');
        document.querySelector(':root').style.setProperty('--main-quater_color', 'rgb(53, 53, 53)');
        document.querySelector(':root').style.setProperty('--filter-main-color-svg', 'invert(0%) sepia(2%) saturate(906%) hue-rotate(227deg) brightness(121%) contrast(100%)');
        document.querySelector(':root').style.setProperty('--filter-second-color-svg', 'invert(100%) sepia(2%) saturate(906%) hue-rotate(227deg) brightness(121%) contrast(100%)');
        signaturePad1.penColor = "black";
    }
}

function input_error(element) {
    var value = element.value.trim();
    //var error_element = element.parentElement.querySelector('.error_container');
    if (element.id == "input_firstName" || element.id == "input_name") {
        if (value != "" && regexFirstLastName.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.id == "input_address") {
        if (value != "") {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.id == "input_siren") {
        if (value != "" && regexSiren.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.id == "input_email" || element.id == "input_customer_email") {
        if (value != "" && regexEmail.test(value)) {
            showHideError(true, element)
        } else {
            showHideError(false, element)
        }
    } else if (element.id == "input_phone_number_customer") {
        element = element.parentElement;
        var error_element = element.parentElement.parentElement.querySelector('.error_container');
        if (value != "" && regexPhoneNumber.test(value)) {
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
    console.log(element);
    var error_element = element.parentElement.querySelector('.error_container');
    if (status) {
        element.classList.remove("input_error");
        element.classList.add("input_valid");
        error_element.style.paddingTop = "0rem";
        error_element.style.height = "0";
    } else {
        element.classList.add("input_error");
        element.classList.remove("input_valid");
        error_element.style.paddingTop = "1.2rem";
        error_element.style.height = "auto";
    }
}

function phoneOnInput(element) {
    var value = element.value.trim();
    console.log(value);
    console.log(value.slice(-2));
    if (!regexNumber.test(value.slice(-1)) || value.length > 14) {
        element.value = value.slice(0, -1);
    } else if (regexTwoNumbers.test(value.slice(-3))) {
        element.value = value.slice(0, -1) + '-' + value.slice(-1);
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
    draggingEle = e.target.parentElement;
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

window.addEventListener('DOMContentLoaded', function () {
    document.querySelectorAll('.svg_devis_line_dots').forEach(dots => {
        dots.addEventListener('mousedown', DragOn);
    })
    document.querySelectorAll('.devis_line_buttons_container').forEach(trash => {
        trash.addEventListener('dragenter', (event) => {
            console.log("prout2");
            //event.target.querySelector('.devis_trash').style.fitler = "invert(12%) sepia(99%) saturate(6471%) hue-rotate(356deg) brightness(110%) contrast(112%);";
            event.target.style.background = "red";

        });
    })
    signaturePad1 = new SignaturePad(document.querySelector("#canvas1"));
    resizeCanvas(document.querySelector("#canvas1"), signaturePad1);
    window.addEventListener("resize", function (e) {
        resizeCanvas(document.querySelector("#canvas1"), signaturePad1);
    });

    document.getElementById('date_emission').valueAsDate = new Date();
    document.getElementById('date_paiement').valueAsDate = new Date();
});

window.addEventListener('mousemove', (event) => {
    cursorX = event.pageX;
    cursorY = event.pageY;
});

function addDevisLine() {
    var firstLine_node = document.getElementById("first_devis_lines_container").firstElementChild;
    var cloneLine_node = firstLine_node.cloneNode(true);
    console.log(cloneLine_node);
    document.getElementById('first_devis_lines_container').appendChild(cloneLine_node);
    console.log(cloneLine_node.getElementsByClassName("svg_devis_line_dots"));
    cloneLine_node.querySelector(".devis_description").innerHTML = "";
    cloneLine_node.querySelector(".devis_quantity").value = "";
    cloneLine_node.querySelector(".devis_price").value = "";
    cloneLine_node.querySelector(".devis_price_total").value = "";
    cloneLine_node.removeAttribute('id');
    cloneLine_node.querySelector(".svg_devis_line_dots").addEventListener('mousedown', DragOn);
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
    parent.querySelector('.devis_price_total').innerHTML = totalLine + "€"

    var totalDevis = 0;
    parent.parentElement.childNodes.forEach((line) => {
        if (line.className == 'devis_line') {
            totalDevis = totalDevis + parseFloat(line.querySelector(".devis_price_total").innerHTML.replace("€", ""));
        }
    })
    document.getElementById("price_indicator_htc").innerHTML = totalDevis + "€";
    document.getElementById("price_indicator_ttc").innerHTML = (totalDevis + (totalDevis * 0.2)) + "€";
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
            //document.getElementById('input_tva').style.height = "auto";
            //document.getElementById('input_tva').style.marginTop = "1rem";
            //document.getElementById('input_tva').style.overflow = "visible";
    
        } else {
            document.getElementById('price_indicator_ttc').style.textDecoration = "line-through";
            document.getElementById('tva_container').style.opacity = "0.5";
            //document.getElementById('input_tva').style.height = "0";
            //document.getElementById('input_tva').style.marginTop = "0";
            //document.getElementById('input_tva').style.overflow = "hidden";
        }
    } else if (element.id == "switch_artisan") {
        if (element.checked) {
            document.getElementById('input_artisan_rm').style.height = "auto";
            document.getElementById('input_artisan_rm').style.marginTop = "1rem";
            document.getElementById('input_artisan_rm').style.overflow = "visible";
        } else {
            document.getElementById('input_artisan_rm').style.height = "0";
            document.getElementById('input_artisan_rm').style.marginTop = "0";
            document.getElementById('input_artisan_rm').style.overflow = "hidden";
        }
        
    } else if (element.id == "switch_commerce") {
        if (element.checked) {
            document.getElementById('input_commercial_rcs').style.height = "auto";
            document.getElementById('input_commercial_rcs').style.marginTop = "1rem";
            document.getElementById('input_commercial_rcs').style.overflow = "visible";
        } else {
            document.getElementById('input_commercial_rcs').style.height = "0";
            document.getElementById('input_commercial_rcs').style.marginTop = "0";
            document.getElementById('input_commercial_rcs').style.overflow = "hidden";
        }
    }
    
}

function generatePDF() {
    // Modify all PDF values with inputs

    //global infos
    var inputDate = new Date(document.getElementById("date_emission").value);
    inputDate = inputDate.toLocaleDateString('en-GB');
    document.getElementById("header_to_generate_date_value").innerHTML = inputDate;

    //worker infos
    document.getElementById("section_to_generate_id_worker_name").innerHTML = document.getElementById("input_firstName").value + " " + document.getElementById("input_lastName").value;
    document.getElementById("section_to_generate_id_worker_address").innerHTML = document.getElementById("input_address").textContent.trim();
    document.getElementById("section_to_generate_id_worker_siren").innerHTML = document.getElementById("input_siren").value;
    if (document.getElementById("toggle_commercant_label").checked) {
        document.getElementById("section_to_generate_id_worker_rcs").style.display = "block";
        document.getElementById("section_to_generate_id_worker_rcs_value") = document.getElementById("input_rcs").value;
    }
    if (document.getElementById("toggle_artisan_label").checked) {
        document.getElementById("section_to_generate_id_worker_rm").style.display = "block";
        document.getElementById("section_to_generate_id_worker_rm_value") = document.getElementById("input_rm").value;
    }

    //customer infos
    document.getElementById("section_to_generate_id_customer_name").innerHTML = document.getElementById("input_customer_name").value;
    document.getElementById("section_to_generate_id_customer_address").innerHTML = document.getElementById("input_customer_address").textContent.trim();
    document.getElementById("section_to_generate_id_customer_siren").innerHTML = document.getElementById("input_customer_siren").value;

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
        if (document.getElementById("switch_tva").checked) {
            document.getElementById("section_to_generate_id_total_ttc").innerHTML = (totalDevis + (totalDevis * 0.2)) + "€";
            document.getElementById("section_devis_to_generate_tva_exempt_text").style.display = "block";
        } else {
            document.getElementById("section_to_generate_id_tva").innerHTML = "0%";
            document.getElementById("section_to_generate_id_total_ttc").innerHTML = totalDevis + "€";
            document.getElementById("section_devis_to_generate_tva_exempt_text").style.display = "none";
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
    if (document.getElementById("input_iban").value.trim() != "" ) {
        document.getElementById("section_to_generate_label_iban").innerHTML = document.getElementById("input_iban").value;
        document.getElementById("section_to_generate_label_iban_name").innerHTML = document.getElementById("input_firstName").value + " " + document.getElementById("input_lastName").value;

    }
    
    //pdf generation
    var element = document.getElementById("absolute_to_generate");
    var signature = document.getElementById('canvas1');
    var signature_width = signature.width / (signature.height/50);
    var pdf = new jsPDF('p', 'pt', 'a4');
    //width 600px * 849px for A4 page
    document.getElementById("absolute_to_generate").style.display = "block";
    pdf.html(element)
        .then(() => {
            pdf.addImage(signature, 'PNG', 60, document.getElementById('section_to_generate_id_retard').offsetTop+document.getElementById('section_to_generate_id_retard').offsetHeight+40, signature_width, 80, "signature", "NONE", 10);
            pdf.setFontSize(12);
            pdf.setTextColor(255,255,255);
            console.log(pdf.getFontList());
            pdf.setFont('Helvetica', 'bold')
            pdf.textWithLink('Payer en ligne', document.getElementById('section_to_generate_internet_paiement_button').offsetLeft+10, getElementOffset(document.getElementById('section_to_generate_internet_paiement_button')).top+16.5, { url: 'http://www.google2.com' });
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