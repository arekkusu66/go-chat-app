const notif_url = `ws://${window.location.host}/notif/ws`;
const notif_ws = new WebSocket(notif_url);


notif_ws.onmessage = (e) => {
    let unreadNotifs = e.data;

    document.querySelector('#notif-count').textContent = unreadNotifs;

    document.querySelector('#notifications').style = 'background-color: rgba(255, 0, 0, 0.5)';
};


async function sendNotif(button) {
    let targetUsername = button.dataset.target;
    let type = button.dataset.type;

    let notification = {
        user: {
            username: targetUsername
        },
        type: type,
    };

    notif_ws.send(JSON.stringify(notification));
};


async function showNotifications() {
    const response = await fetch('/notifications?action=get', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json'
        },
    });

    const notif_c = document.querySelector('#notif-c');
    const notif_count = document.querySelector('#notif-count');

    notif_c.innerHTML = '';

    const notifications = await response.json();

    if (notifications.length === 0) {
        notif_c.innerHTML = '<b>no notifications</b>';
        notif_count.textContent = '';
        return
    };
    

    notifications.map((notif) => {
        notif_c.innerHTML += `
        <div id="notif-${notif.id}" style="${notif.read ? 'background-color:rgba(128, 128, 128, 0.5)' : ''}">
            <h4>${formatDate(notif.date)}</h4>
            <a href="${notif.link}">${notif.message}</a>
            <button data-id="${notif.id}" data-action="mark-as-read" onclick="notifAction(this)">Mark as read</button>
            <button data-id="${notif.id}" data-action="delete" onclick="notifAction(this)">Delete</button>
        </div>`;
    });
};


async function notifAction(button) {
    const id = button.dataset.id;
    const action = button.dataset.action;

    const notifs = document.querySelector('#notifications');
    const notif = document.querySelector(`#notif-${id}`);
    const notif_count = document.querySelector('#notif-count');
    
    try {

        if (action === 'mark-as-read') {
            await fetch(`/notifications?id=${id}&action=${action}`, {method: 'POST'});
            notif.style.backgroundColor = 'rgba(128, 128, 128, 0.5)';
        } else if (action === 'delete') {
            await fetch(`/notifications?id=${id}&action=${action}`, {method: 'POST'});
            notif.remove();
        };

        const response = await fetch('/notifications?action=are-there-unread-notifs', {method: 'POST'});
        const areThereUnreadNotifs = await response.text();

        if (areThereUnreadNotifs === 'false') {
            notifs.style.backgroundColor = '';
            notif.style.backgroundColor = '';
            notif_count.textContent = '';
        };

    } catch(error) {
        console.log(error);
    };
};


notif_ws.onopen = () => {
    console.log('connected');
};


notif_ws.onerror = (error) => {
    alert(error);
};