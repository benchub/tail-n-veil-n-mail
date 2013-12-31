<?php
$days = 1;
$bucket = " is null";

if(isset($_GET["days"]))
{
        if(is_numeric($_GET["days"]))
        {
                if($_GET["days"] > 0)
                {
                        $days = $_GET["days"];
                }
        }
}

if(isset($_GET["id"]))
{
        if(is_numeric($_GET["id"]))
        {
                if($_GET["id"] > 0)
                {
                        $bucket = " = " . $_GET["id"];
                }
        }
}

// Connecting, selecting database
$dbconn = pg_connect("host=blahblah dbname=tnvnm user=www password=blahblah")
    or die('Could not connect: ' . pg_last_error());


if (isset($_GET["n"]))
{
        $eventQuark = "normalize_query(event)";
}
else
{
        $eventQuark = "event";
}
$query = "SELECT host,finished,$eventQuark from events where finished > now()-interval '1 day'*$days and bucket_id $bucket order by finished desc limit 500";
$events = pg_query($query) or die('Query failed: ' . pg_last_error());

echo "<h2>Showing the " . pg_num_rows($events) . " instances of ";

if ($bucket == " is null")   
{
        echo "interesting things";
}
else
{
        $query = "SELECT name from buckets where id $bucket";
        $bucket_info = pg_query($query) or die('Query failed: ' . pg_last_error());
        while ($row = pg_fetch_row($bucket_info))
        {
                echo $row[0];
        }
        pg_free_result($bucket_info);
}

echo " over the last $days days</h2>";

$me = "<a href='bucketDetails.php?days=$days";
if ($bucket != " is null")   
{
        $me = $me . "&id=" . $_GET["id"];
}

if (isset($_GET["n"]))
{
        echo "[ $me'>Show queries as they were</a> | Showing queries normalized ]";
}
else
{
        echo "[ Showing queries as they were | $me&n=1'>Show queries normalized</a> ]";
}

echo "<hr><table>\n";
    echo "\t<tr>\n";
        echo "\t\t<td>When</td>\n";
        echo "\t\t<td>Where</td>\n";
        echo "\t\t<td>What</td>\n";
    echo "\t</tr>\n";
while ($row = pg_fetch_row($events))
{
    echo "\t<tr>\n";
        echo "\t\t<td>" . htmlentities($row[1], ENT_QUOTES) . "</td>\n";
        echo "\t\t<td>" . htmlentities($row[0], ENT_QUOTES) . "</td>\n";
        echo "\t\t<td><pre>" . $row[2] . "</pre></td>\n";
    echo "\t</tr>\n";
}
echo "</table><hr>\n";

// Free resultset
pg_free_result($events);

// Closing connection
pg_close($dbconn);
?>
