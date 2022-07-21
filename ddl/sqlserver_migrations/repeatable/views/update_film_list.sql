CREATE OR ALTER TRIGGER update_film_list ON film_list INSTEAD OF UPDATE AS
BEGIN
    DECLARE
        -- INSERTED
        @inserted_fid INT
        ,@inserted_title NVARCHAR(255)
        ,@inserted_description NVARCHAR(MAX)
        ,@inserted_category NVARCHAR(45)
        ,@inserted_price DECIMAL(4,2)
        ,@inserted_length INT
        ,@inserted_rating NVARCHAR(255)
        ,@inserted_actors NVARCHAR(MAX)
        -- DELETED
        ,@deleted_fid INT
        ,@deleted_title NVARCHAR(255)
        ,@deleted_description NVARCHAR(MAX)
        ,@deleted_category NVARCHAR(45)
        ,@deleted_price DECIMAL(4,2)
        ,@deleted_length INT
        ,@deleted_rating NVARCHAR(255)
        ,@deleted_actors NVARCHAR(MAX)
    ;

    SELECT
        @inserted_fid = INSERTED.fid
        ,@inserted_title = INSERTED.title
        ,@inserted_description = INSERTED.description
        ,@inserted_category = INSERTED.category
        ,@inserted_price = INSERTED.price
        ,@inserted_length = INSERTED.length
        ,@inserted_rating = INSERTED.rating
        ,@inserted_actors = INSERTED.actors
    FROM
        INSERTED
    ;

    SELECT
        @deleted_fid = DELETED.fid
        ,@deleted_title = DELETED.title
        ,@deleted_description = DELETED.description
        ,@deleted_category = DELETED.category
        ,@deleted_price = DELETED.price
        ,@deleted_length = DELETED.length
        ,@deleted_rating = DELETED.rating
        ,@deleted_actors = DELETED.actors
    FROM
        DELETED
    ;

    IF @inserted_fid <> @deleted_fid
        THROW 50001, 'You are not allowed to update film_list.fid', 1;

    IF @inserted_category <> @deleted_category
        THROW 50001, 'You are not allowed to update film_list.category', 1;

    IF @inserted_actors <> @deleted_actors
        THROW 50001, 'You are not allowed to update film_list.actors', 1;

    IF @inserted_title <> @deleted_title
        OR @inserted_description <> @deleted_description
        OR @inserted_price <> @deleted_price
        OR @inserted_length <> @deleted_length
        OR @inserted_rating <> @deleted_rating
    BEGIN
        UPDATE film
        SET
            title = @inserted_title
            ,description = @inserted_description
            ,rental_rate = @inserted_price
            ,length = @inserted_length
            ,rating = @inserted_rating
        WHERE
            film_id = @inserted_fid
        ;
    END;
END;
