new Vue({
    el: '#app',
    data: {
        activeNames: ['first'], // 控制折叠面板的打开状态
        link: '', // 用户输入的链接
        result: '', // 显示处理结果或错误消息
        isEnglish: false // Add a data property to track language state
    },
    methods: {
        // 验证输入的 URL 是否有效
        isValidUrl(url) {
            const urlRegex = /http[s]?:\/\/[^\s]+/;
            return urlRegex.test(url);
        },
        // 检查 URL 是否属于支持的平台（网易云或QQ音乐）
        isSupportedPlatform(url) {
            const supportedPlatformsRegex = /(163)|(qq)/;
            return supportedPlatformsRegex.test(url);
        },

        // 异步函数，用于获取链接的详细信息
        async fetchLinkDetails() {
            this.link = this.link.trim(); // 去除输入链接的首尾空白字符

            // 检查链接是否有效
            if (!this.isValidUrl(this.link)) {
                this.result = '❌链接无效，请输入有效的网址。';
                return;
            }

            // 检查链接是否属于支持的平台
            if (!this.isSupportedPlatform(this.link)) {
                this.result = '❌不支持的平台，请输入网易云或QQ音乐歌单链接。';
                return;
            }

            // 准备发送请求的参数
            const params = new URLSearchParams();
            params.append('url', this.link);

            try {
                // 发送 POST 请求并处理响应
                const response = await axios.post('https://music.unmeta.cn/songlist', params, {
                    headers: {'Content-Type': 'application/x-www-form-urlencoded'}
                });
                // 设置结果，如果响应中有数据则显示歌曲列表，否则显示失败信息
                this.result = response.data ? response.data.data.songs.join('\n') : '请求失败，请稍后再试~';
            } catch (error) {
                // 如果请求过程中发生错误，记录错误并显示错误消息
                console.error(error);
                this.result = '❌请求过程中发生错误，请稍后重试~';
            }
        },
        // 复制结果到剪贴板的方法
        copyResult() {
            const textarea = document.createElement('textarea'); // 创建一个临时的 textarea 元素
            textarea.value = this.result; // 设置 textarea 的值为结果文本
            document.body.appendChild(textarea); // 将 textarea 添加到文档中
            textarea.select(); // 选中 textarea 的内容
            document.execCommand('copy'); // 执行复制命令
            document.body.removeChild(textarea); // 从文档中移除 textarea
            this.$message.success('已复制到剪贴板'); // 显示复制成功的消息
        },
        toggleLanguage() {
            this.isEnglish = !this.isEnglish;

            const headerElement = this.$el.querySelector('.el-header');
            const buttonElement = this.$el.querySelector('.header-layout .el-button');
            const inputElement = this.$el.querySelector('.el-input__inner');
            const textareaElement = this.$el.querySelector('.el-textarea__inner');
            const collapseItemHeaderElement = this.$el.querySelector('.el-collapse-item__header');
            const collapseItemElements = this.$el.querySelectorAll('.el-collapse-item ol li');
            const collapseElement = this.$el.querySelector('.el-collapse');
            const songListButtonElement = this.$el.querySelector('.lang-song-list-btn');
            const copyButtonElement = this.$el.querySelector('.lang-copy-btn');

            if (this.isEnglish) {
                headerElement.textContent = 'Migrate NetEase Cloud / QQ Music Playlists to Apple / Youtube / Spotify Music';
                buttonElement.textContent = '中文';
                inputElement.setAttribute('placeholder', 'Enter any playlist link, such as: http://163cn.tv/zoIxm3');
                textareaElement.setAttribute('placeholder', 'The result will be displayed here');
                collapseItemHeaderElement.textContent = '《User Guide》';
                collapseItemElements[0].textContent = 'Enter the playlist link, such as: http://163cn.tv/zoIxm3';
                collapseItemElements[1].textContent = 'Copy the query result';
                collapseItemElements[2].innerHTML = 'Open <b><a href="https://www.tunemymusic.com/zh-CN/transfer">TunemyMusic</a></b> website';
                collapseItemElements[3].textContent = 'Select the playlist source "Any Text", paste the playlist just copied, select Apple/Youtube/Spotify Music as the destination, and confirm the migration';
                songListButtonElement.textContent = 'Get Song List';
                copyButtonElement.textContent = 'Copy Result';
            } else {
                headerElement.textContent = '迁移网易云/QQ音乐歌单至 Apple/Youtube/Spotify Music';
                buttonElement.textContent = 'English';
                inputElement.setAttribute('placeholder', '输入任意歌单链接，如：http://163cn.tv/zoIxm3');
                textareaElement.setAttribute('placeholder', '结果会显示在这里');
                collapseItemHeaderElement.textContent = '《使用指南》';
                collapseItemElements[0].textContent = '输入歌单链接，如：http://163cn.tv/zoIxm3';
                collapseItemElements[1].textContent = '复制结果';
                collapseItemElements[2].innerHTML = '打开 <b><a href="https://www.tunemymusic.com/zh-CN/transfer">TunemyMusic</a></b> 网站';
                collapseItemElements[3].textContent = '选择歌单来源“任意文本”，将刚刚复制的歌单粘贴进去，选择 Apple/Youtube/Spotify Music 作为目的地，确认迁移';
                songListButtonElement.textContent = '获取歌单';
                copyButtonElement.textContent = '复制结果';
            }
        }

    }
});