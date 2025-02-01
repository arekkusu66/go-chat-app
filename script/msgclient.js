const msg_url = new URL(window.location.href);
const id = msg_url.pathname.match(/\/chat\/(\d+)/)[1];

const wsurl = `ws://${window.location.host}/msg/ws/` + id;
const msg_ws = new WebSocket(wsurl);

const delUrl = `ws://${window.location.host}/msg/ws/del/` + id;
const delws = new WebSocket(delUrl);


msg_ws.onopen = () => {
    console.log('connected');
};


msg_ws.onerror = (error) => {
    alert(error);
};


msg_ws.onmessage = (e) => {
    let message = JSON.parse(e.data);

    addMessage(message);
};


function sendMsg() {
    const message = document.querySelector('#send');
    const replyId = document.querySelector('#id-reply');

    const id = msg_url.pathname.match(/\/chat\/(\d+)/)[1];
    
    let datas = {
        text: message.value,
        chatRoomId: parseInt(id),
        replyId: parseInt(replyId.textContent),
        replyStatus: 'not_deleted',
    };

    msg_ws.send(JSON.stringify(datas));

    message.value = '';
    replyId.textContent = '';
};


async function addMessage(message) {
    const messageDatasResponse = await fetch(`/get/datas?id=${message.id}`, {
        method: 'POST'
    });

    const messageDatas = await messageDatasResponse.json();

    const chatOP = `<h5 style="color:${message.userId === message.chatOp.String ? 'blue' : 'rgb(177, 6, 6)'}">${message.user.username}${message.userId === message.chatOp.String ? ' - OP' : ''}</h5>`;

    const getOptions = `<div><button data-message-id="${message.id}" id="options-get-${message.id}">options</button></div>`;
    const options = `<div id="options-${message.id}" class="msg-options"></div>`;

    const replies = `${message.replyId !== 0 && message.reply ? (`<div id="reply-${message.replyId}"><a href="#message-${message.replyId}">reply to ${message.reply.text}</a></div>`) : ''}`;

    const msgDatas = `<h3 id="message-${message.id}"></h3><i>at ${formatDate(message.date)}</i><hr />`;

    let notBlocked = `
        <div ${messageDatas.isReply ? `style="background-color:rgba(238, 238, 0, 0.5)"` : ''} id="message-c-${message.id}">
            ${chatOP}
            ${getOptions}
            ${options}
            ${replies}
            ${msgDatas}
        </div>
    `;

    let blocked = `
        <div id="message-c-${message.id}">
            <p>Message from a blocked user<button onclick="showMessage(this)" data-id="${message.id}">Show</button></p>
        </div>
    `;

    document.querySelector('#messages').innerHTML += messageDatas.isBlocked ? blocked : notBlocked;


    document.querySelector(`#message-${message.id}`).textContent = message.text;

    
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


delws.onerror = (error) => {
    alert(error);
};


delws.onmessage = (e) => {
    let id = e.data;

    document.querySelector(`#message-c-${id}`).remove();

    Array.from(document.querySelectorAll(`#reply-${id}`)).map((e) => e.innerHTML = `<div><i style="color:red">Reply to deleted message</i></div>`);
};