window.onload = function() {
	let updateElm = document.createElement('div');
	updateElm.innerHTML = document.body.dataset["update"];

	document.body.insertBefore(updateElm, document.body.firstChild);
}
