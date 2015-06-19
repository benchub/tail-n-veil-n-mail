<?php
$dbconn = pg_connect("host=your.db.hostname dbname=tnvnm user=www password=SuperSekritPassword")
    or die('Could not connect: ' . pg_last_error());
?>
