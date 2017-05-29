import QtQuick 2.5
import QtQuick.Window 2.2

Window {
    visible: true
    width: 600
    height: 400

    MouseArea {
        anchors.fill: parent
        onClicked: {
            Qt.quit();
        }
    }

    Image {
        source: "qrc:/delete.svg"
        sourceSize.width: 400
        sourceSize.height: 400
        anchors.centerIn: parent
    }

    Text {
        text: qsTr("Hello World") + magicNumber
        anchors.centerIn: parent
        color: "#ffffff"
    }

    Timer {
        interval: 2000
        repeat: false
        onTriggered: Qt.quit()
        running: true
    }
}
