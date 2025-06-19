function scanTelnet() {
    const ipRange = document.getElementById('ipRange').value;
    const port = document.getElementById('port').value;

    if (!ipRange || !port) {
        alert('请输入 IP 地址段和端口号');
        return;
    }

    // 清空之前的检测结果
    document.getElementById('telnetResultsBody').innerHTML = '';

    // 发送请求到后端
    fetch(`/scan?ipRange=${encodeURIComponent(ipRange)}&port=${encodeURIComponent(port)}`)
        .then(response => response.json())
        .then(data => {
            const resultsBody = document.getElementById('telnetResultsBody');
            data.forEach(item => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>${item.ip}</td>
                    <td>${item.port}</td>
                    <td class="${item.isOpen ? 'success' : 'failure'}">
                        ${item.isOpen ? '开放' : '未开放'}
                    </td>
                    <td>${item.isOpen ? item.uri : '-'}</td>
                `;
                resultsBody.appendChild(row);
            });
        })
        .catch(error => {
            console.error('Error:', error);
            alert('请求失败，请检查输入或联系管理员');
        });
}

function scanPing() {
    const ipRange = document.getElementById('ipRange').value;

    if (!ipRange) {
        alert('请输入 IP 地址段');
        return;
    }

    // 清空之前的检测结果
    document.getElementById('pingResultsBody').innerHTML = '';

    // 发送请求到后端
    fetch(`/scanPing?ipRange=${encodeURIComponent(ipRange)}`)
        .then(response => response.json())
        .then(data => {
            const resultsBody = document.getElementById('pingResultsBody');
            data.forEach(item => {
                const row = document.createElement('tr');
                row.innerHTML = `
                    <td>${item.ip}</td>
                    <td class="${item.isReachable ? 'success' : 'failure'}">
                        ${item.isReachable ? '可达' : '不可达'}
                    </td>
                `;
                resultsBody.appendChild(row);
            });
        })
        .catch(error => {
            console.error('Error:', error);
            alert('请求失败，请检查输入或联系管理员');
        });
}
