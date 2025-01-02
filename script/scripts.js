function sendDatas() {
    const title = document.querySelector('#chatroom-title')

    fetch('/create/chat', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
        body: JSON.stringify({
            title: title.value
        })
    }).catch((err) => console.log(err));

    title.value = '';
};


async function changeUsername() {
    const newUsername = document.querySelector('#new-username');

    try {
        let existsResponse = await fetch(`/exists/${newUsername.value}`, {method:'POST'});

        if (!existsResponse.ok) {
            alert('this username already exists!');
            throw new Error('username already in use!');
        };
    
        await fetch(`/edit/username?username=${newUsername.value}`, {method:'POST'});
    } catch (err) {
        console.log(err);
        return;
    };
};


function editDescription() {
    const description = document.querySelector('#write-description');

    fetch(`/edit/profile?edit=description`, {
        method: 'POST',
        headers: {
            'Content-Type': 'text/plain'
        },
        body: description.value

    }).then((res) => res.text()).then((res) => document.querySelector('#description').textContent = res).catch((err) => {console.log(err); return});
};


async function getOptions(button) {
    const id = button.dataset.id;

    const response = await fetch(`/get/options?id=${id}`, {
        method: 'POST'
    });

    const htmlDatas = await response.text();

    const element = document.querySelector(`#options-${id}`);

    element.innerHTML = htmlDatas;
};


function cancelOptions() {
    document.querySelector('#id-reply').textContent = '';
    Array.from(document.querySelectorAll('.msg-options')).map((element) => element.innerHTML = '');
};


function reply(button) {
    const id = button.dataset.id;
    const idReply = document.querySelector('#id-reply');
    idReply.textContent = id;
};


function deleteMsg(button) {
    const id = button.dataset.id;
    delws.send(id);
};


function formatDate(inputString) {
    const dateTimeParts = inputString.match(/(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2}\.\d+)/);
    
    if (!dateTimeParts) {
      throw new Error('Invalid input format');
    }
  
    const [_, year, month, day, hours, minutes, seconds] = dateTimeParts;
  
    const date = new Date(year, month - 1, day, hours, minutes, seconds);
  
    return `${date.getDate().toString().padStart(2, '0')}.${(date.getMonth() + 1).toString().padStart(2, '0')}.${date.getFullYear()} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
};