function sendDatas() {
    const title = document.querySelector('#chatroom-title')

    fetch(`/create/chat?title=${title.value}`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        }
    }).catch((err) => console.log(err));

    title.value = '';
};


async function changeUsername() {
    const newUsername = document.querySelector('#new-username');

    try {
        await fetch(`/edit/username?username=${newUsername.value}`, {method:'POST'});

    } catch (err) {
        alert(err);
        console.log(err);
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


async function showMessage(button) {
    const id = button.dataset.id;

    try {
        const response = await fetch(`/get/message?id=${id}`, {
            method: 'POST'
        });

        const datas = await response.json();

        console.log(datas);

        const hide = `<button onclick="hideMessage(this)" data-id="${id}">Hide</button>`;

        const chatOP = `
            <h5 
                style="color:${datas.message.user_id === datas.chatroom.creator_id ? 'blue' : 'rgb(177, 6, 6)'}">
                
                ${datas.user.username}${datas.message.user_id === datas.chatroom.creator_id ? ' - OP' : ''}
            </h5>`;

        const replies = `
            ${datas.message.reply_id.Valid && datas.reply_text.Valid ? 
            
            `<div id="reply-${datas.message.reply_id}">
                <a href="#message-${datas.message.reply_id.Int64}">reply to ${datas.reply_text.String}</a>
            </div>` : ''}`;

        const msgDatas = `<h3 id="message-${id}"></h3><i>at ${formatDate(datas.message.date)}</i><hr />`;    

        document.querySelector(`#message-c-${id}`).innerHTML = `
            ${hide}
            ${chatOP}
            ${replies}
            ${msgDatas}
        `;

        document.querySelector(`#message-${id}`).textContent = datas.message.text;

    } catch (error) {
        console.log(error);
        return;
    };
};


function hideMessage(button) {
    const id = button.dataset.id;

    document.querySelector(`#message-c-${id}`).innerHTML = `
        <div>
            <p>Message from a blocked user<button onclick="showMessage(this)" data-id="${id}">Show</button></p>
        </div>
    `;
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


function formatDate(inputString) {
    const dateTimeParts = inputString.match(/(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2}\.\d+)/);
    
    if (!dateTimeParts) {
      throw new Error('Invalid input format');
    }
  
    const [_, year, month, day, hours, minutes, seconds] = dateTimeParts;
  
    const date = new Date(year, month - 1, day, hours, minutes, seconds);
  
    return `${date.getDate().toString().padStart(2, '0')}.${(date.getMonth() + 1).toString().padStart(2, '0')}.${date.getFullYear()} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
};