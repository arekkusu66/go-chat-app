const url = new URL(window.location.href);
const id = url.pathname.match(/\/dm\/(\d+)/)[1];

const wsurl = `ws://${window.location.host}/dm/ws/` + id;
const ws = new WebSocket(wsurl);

const delUrl = `ws://${window.location.host}/dm/ws/del/` + id;
const delws = new WebSocket(delUrl);


function sendMsg() {
    const message = document.querySelector('#send');
    const replyId = document.querySelector('#id-reply');

    const id = url.pathname.match(/\/dm\/(\d+)/)[1];

    let datas = {
        text: message.value,
        dmId: parseInt(id),
        replyId: parseInt(replyId.textContent),
        replyStatus: 'not_deleted',
    };

    ws.send(JSON.stringify(datas));

    message.value = '';
    replyId.textContent = '';
};


ws.onopen = () => {
    console.log('connected');
};


ws.onerror = (error) => {
    alert(error);
};


ws.onmessage = (e) => {
    let message = JSON.parse(e.data);

    addMessage(message);
};



async function addMessage(message) {

    const response = await fetch(`/get/datas?id=${message.replyId}`, {
        method: 'POST'
    });

    const isReply = await response.text();


    const username = `<h5 style="color:green">${message.user.username}</h5>`;

    const getOptions = `<div><button data-message-id="${message.id}" id="options-get-${message.id}">options</button></div>`;
    const options = `<div id="options-${message.id}" class="msg-options"></div>`;

    const replies = `${message.replyId !== 0 && message.reply ? (`<div id="reply-${message.replyId}"><a href="#message-${message.replyId}">reply to ${message.reply.text}</a></div>`) : ''}`;

    const msgDatas = `<h3 id="message-${message.id}"></h3><i>at ${message.date}</i><hr />`;


    document.querySelector('#messages').innerHTML += `
    <div ${isReply === 'true' ? `style="background-color:rgba(238, 238, 0, 0.5)"` : ''} id="message-c-${message.id}">
        ${username}
        ${getOptions}
        ${options}
        ${replies}
        ${msgDatas}
    </div>`;


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