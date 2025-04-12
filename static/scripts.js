let isEditing = false;

function selectCharacter(element) {
    if (isEditing) {
        const confirmLeave = confirm("You have unsaved changes. Do you want to leave without saving?");
        if (!confirmLeave) {
            return;
        }
        isEditing = false;
    }

    fetch('/select-character', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({ id: parseInt(element.getAttribute("data-id")) }),
    }).then(response => response.text())
        .then(html => {
            document.getElementById('character-list').innerHTML = html;
            initializeSortable();
        })
        .catch((error) => console.error('Error:', error));
}

function editCharacter(button, event) {
    event.stopPropagation();
    isEditing = true;
    const characterDiv = button.closest('.character');
    characterDiv.querySelector('.edit-mode').style.display = '';
    characterDiv.querySelector('.view-mode').style.display = 'none';
}

function saveCharacter(button) {
    isEditing = false;
    const characterDiv = button.closest('.character');
    const id = parseInt(characterDiv.getAttribute("data-id"));
    const name = characterDiv.querySelector('input[name="name"]').value;
    const armorClass = parseFloat(characterDiv.querySelector('input[name="armorClass"]').value);
    const maxHP = parseFloat(characterDiv.querySelector('input[name="maxHP"]').value);
    const currentHP = parseFloat(characterDiv.querySelector('input[name="currentHP"]').value);
    const initiative = parseFloat(characterDiv.querySelector('input[name="initiative"]').value);

    if (name === '') {
        alert('Name is required');
        return;
    }

    fetch('/save-character', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
        },
        body: JSON.stringify({id, name, armorClass, maxHP, currentHP, initiative}),
    }).then(response => {
        if (!response.ok) {
            throw new Error('Failed to save character');
        }
        return response.text();
    }).then(html => {
        characterDiv.outerHTML = html;
    }).catch(error => {
        alert(error.message);
    });
}

function cancelEdit(button) {
    isEditing = false;
    const characterDiv = button.closest('.character');
    characterDiv.querySelector('.edit-mode').style.display = 'none';
    characterDiv.querySelector('.view-mode').style.display = '';
}

function stopPropagation(event) {
    event.stopPropagation();
}

function initializeSortable() {
    new Sortable(document.getElementById('character-list'), {
        animation: 150,
        ghostClass: 'sortable-ghost',
        onEnd: function (evt) {
            fetch('/reorder', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ oldIndex: evt.oldIndex, newIndex: evt.newIndex }),
            }).then(response => response.json())
                .then(data => console.log('Success:', data))
                .catch((error) => console.error('Error:', error));
        }
    });
}