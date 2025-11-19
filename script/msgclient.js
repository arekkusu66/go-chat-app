const msgurl = new URL(window.location.href);
const path = msgurl.pathname.match(/\/(chat|dm)\/(\d+)/);
const type = path[1];
const id = path[2];

const wsurl = `ws://${window.location.host}/msg/ws/` + id;
const msgws = new WebSocket(wsurl);


msgws.onopen = () => {
    console.log('connected');
};


msgws.onerror = (error) => {
    alert(error);
};


msgws.onmessage = (e) => {
    let message = JSON.parse(e.data);

    switch (message.type) {
        case 'MSG':
            addMessage(message.data);
            break;

        case 'DEL':
            document.querySelector(`#message-c-${message.data.id}`).remove();

            Array.from(document.querySelectorAll(`#reply-${message.data.id}`))
                .forEach(e => e.innerHTML = `<div><i style="color:red">Reply to a deleted message</i></div>`);
            
            break;

        default:
            break;
    };
};


function sendMsg() {
    const message = document.querySelector('#send');
    const reply_id = document.querySelector('#id-reply');

    if (message.value === '') {
        return;
    };
    
    let msg = {
        type: 'MSG',
        data: {
            text: message.value,
            reply_status: 'no_reply',
        },
    };

    msg.data.reply_id = reply_id.textContent != '' ? 
        {Int64: parseInt(reply_id.textContent), Valid: true} : null;

    switch (type) {
        case 'chat':
            msg.data.chatroom_id = {Int64: parseInt(id), Valid: true};
            break;
        case 'dm':
            msg.data.dm_id = {Int64: parseInt(id), Valid: true};
            break;
        default:
            break;
    };

    msgws.send(JSON.stringify(msg));

    message.value = '';
    reply_id.textContent = '';
};


async function addMessage(message) {
    const resExtraDatas = await fetch(`/get/message?type=0&id=${message.id}`, {
        method: 'POST'
    });

    const extraDatas = await resExtraDatas.json();

    let chatordm = message.chatroom_id.Valid ? 
    
        (`style="color:${message.user_id === message.creator_id ? 'blue' : 'rgb(177, 6, 6)'}">
        ${message.username}${message.user_id === message.creator_id ? ' - OP' : ''}`) :
        
        (`style="color:rgb(177, 6, 6)">
        ${message.username}`);

    const userDatas = `
        <h5 ${chatordm} </h5>`;

    const getOptions = `
        <div>
            <button data-message-id="${message.id}" id="options-get-${message.id}">options</button>
        </div>`;

    const options = `<div id="options-${message.id}" class="msg-options"></div>`;

    const replies = `
        ${(message.reply_id.Valid && message.reply_text.Valid) ? 
            `<div id="reply-${message.reply_id.Int64}">
                <a href="#message-${message.reply_id.Int64}">reply to ${message.reply_text.String}</a>
            </div>` : ''}`;

    const msgDatas = `
        <h3 id="message-${message.id}">${message.text}</h3>
        <i>at ${formatDate(message.date)}</i><hr />`;

    let notBlocked = `
        <div 
            ${extraDatas.is_reply ? `style="background-color:rgba(255, 255, 0, 0.5)"` : ''}
            id="message-c-${message.id}">
            ${userDatas}
            ${getOptions}
            ${options}
            ${replies}
            ${msgDatas}
        </div>
    `;

    let blocked = `
        <div id="message-c-${message.id}">
            <p>Message from a blocked user
                <button onclick="showMessage(this)" data-id="${message.id}">Show</button>
            </p>
        </div>
    `;

    document.querySelector('#messages').innerHTML += extraDatas.is_blocked ? blocked : notBlocked;

    if (!extraDatas.is_blocked) {
        document.querySelector(`#message-${message.id}`).textContent = message.text;
    };

    
    document.querySelector('#messages').addEventListener('click', async (event) => {
        if (event.target.matches(`button[data-message-id]`)) {
            const messageId = event.target.getAttribute('data-message-id');
            const response = await fetch(`/get/options?id=${messageId}`, {
                method: 'POST'
            });
        
            const htmlDatas = await response.text();
        
            const element = document.querySelector(`#options-${messageId}`);
            element.innerHTML = htmlDatas;
        }
    });
};


function deleteMsg(button) {
    const btnID = button.dataset.id;

    let msg = {
        type: 'DEL',
        data: {id: parseInt(btnID)},
    };

    switch (type) {
        case 'chat':
            msg.data.chatroom_id = {Int64: parseInt(id), Valid: true};
            break;
        case 'dm':
            msg.data.dm_id = {Int64: parseInt(id), Valid: true};
            break;
        default:
            break;
    };

    msgws.send(JSON.stringify(msg));
};